package bridge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"MusicLe/internal/browser"
	"MusicLe/state"
)

const mp3Ext = ".mp3"

var allowedAudioExt = map[string]bool{
	".mp3": true,
}

// ImportFromBrowser writes a browser-extracted playlist (tracks with their
// download source) into the profile's playlist directory as song_list.txt
// metadata. The audio files are not downloaded here.
func ImportFromBrowser(profileFolder, plFolder, plName string, tracks []browser.Track) error {
	if err := state.Current.CreatePlaylistStructure(profileFolder, plFolder, plName, "", ""); err != nil {
		return err
	}
	listPath := state.Current.SongListPath(profileFolder, plFolder)
	for i := range tracks {
		t := &tracks[i]
		name := trackBaseName(t)
		if err := state.AppendSongSource(listPath, name+mp3Ext, t.Title, t.Artist, t.Duration, t.Source); err != nil {
			return err
		}
	}
	return nil
}

// trackBaseName produces a stable file base name for an imported track.
func trackBaseName(t *browser.Track) string {
	if t.VideoID != "" {
		return t.VideoID
	}
	return sanitizeName(t.Title + " - " + t.Artist)
}

func sanitizeName(name string) string {
	name = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		}
		return r
	}, name)
	return strings.TrimSpace(name)
}

// addLocalFile imports an mp3 file or directory into the playlist directory.
func addLocalFile(sourcePath, playlistDir string) *Result {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return &Result{Status: "error", Error: fmt.Sprintf("file not found: %s", sourcePath)}
	}

	if info.IsDir() {
		return importDirectory(sourcePath, playlistDir)
	}
	return importSingleFile(sourcePath, playlistDir)
}

func importDirectory(sourceDir, playlistDir string) *Result {
	// Validate all audio files are mp3
	var nonMP3 []string
	filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != mp3Ext {
			nonMP3 = append(nonMP3, info.Name())
		}
		return nil
	})

	if len(nonMP3) > 0 {
		limit := len(nonMP3)
		if limit > 5 {
			limit = 5
		}
		errMsg := "Non-mp3 files found:\n" + strings.Join(nonMP3[:limit], "\n")
		return &Result{Status: "error", Error: errMsg}
	}

	var imported int
	var errors []string
	filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) != mp3Ext {
			return nil
		}
		r := importSingleFile(path, playlistDir)
		if r.Status == "ok" {
			imported++
		} else {
			errors = append(errors, info.Name()+": "+r.Error)
		}
		return nil
	})

	msg := fmt.Sprintf("Imported %d file(s)", imported)
	if len(errors) > 0 {
		msg += fmt.Sprintf(", %d error(s)", len(errors))
	}
	return &Result{
		Status:   "ok",
		Filename: msg,
		Title:    msg,
		Duration: 0,
	}
}

func importSingleFile(sourcePath, playlistDir string) *Result {
	ext := strings.ToLower(filepath.Ext(sourcePath))
	if ext != mp3Ext {
		return &Result{Status: "error", Error: fmt.Sprintf("unsupported format: %s (only mp3 allowed)", ext)}
	}

	if err := os.MkdirAll(playlistDir, 0755); err != nil {
		return &Result{Status: "error", Error: fmt.Sprintf("create playlist dir: %v", err)}
	}

	basename := filepath.Base(sourcePath)
	if !strings.HasSuffix(strings.ToLower(basename), mp3Ext) {
		return &Result{Status: "error", Error: "only mp3 files are supported"}
	}

	destPath := filepath.Join(playlistDir, basename)

	// Handle name collisions
	counter := 1
	for {
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			break
		}
		name := strings.TrimSuffix(basename, mp3Ext)
		destPath = filepath.Join(playlistDir, fmt.Sprintf("%s_%d.mp3", name, counter))
		counter++
	}

	// Copy file
	input, err := os.ReadFile(sourcePath)
	if err != nil {
		return &Result{Status: "error", Error: fmt.Sprintf("read source: %v", err)}
	}
	if err := os.WriteFile(destPath, input, 0644); err != nil {
		return &Result{Status: "error", Error: fmt.Sprintf("write dest: %v", err)}
	}

	// Extract metadata
	meta := extractMetadata(destPath)
	if meta.Status == "error" {
		meta = &Result{
			Status: "ok",
			Title:  strings.TrimSuffix(filepath.Base(destPath), mp3Ext),
			Artist: "Unknown",
		}
	}

	durStr := fmtDuration(meta.Duration)
	listPath := filepath.Join(playlistDir, "song_list.txt")
	if err := state.AppendSong(listPath, filepath.Base(destPath), meta.Title, meta.Artist, durStr); err != nil {
		return &Result{Status: "error", Error: fmt.Sprintf("append song: %v", err)}
	}

	return &Result{
		Status:   "ok",
		Filename: filepath.Base(destPath),
		Title:    meta.Title,
		Artist:   meta.Artist,
		Duration: meta.Duration,
		ArtPath:  meta.ArtPath,
	}
}

func fmtDuration(seconds float64) string {
	s := int(seconds)
	return fmt.Sprintf("%02d:%02d", s/60, s%60)
}
