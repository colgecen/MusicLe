package bridge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"MusicLeCLI/bridge/download/music"
	"MusicLeCLI/bridge/download/playlist"
	"MusicLeCLI/internal/engine"
	"MusicLeCLI/state"
)

// engineDisabled reports whether the user opted out of the pure-Go engine
// for this run (via env var). Useful for debugging the legacy pipeline.
func engineDisabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("MUSICLE_NO_ENGINE")))
	return v == "1" || v == "true" || v == "yes"
}

// isSpotifyURL heuristically detects Spotify links/URIs so the unified
// engine dispatcher can route them through yt-dlp (which resolves Spotify
// pages directly), kept separate from plain YouTube URLs.
func isSpotifyURL(url string) bool {
	u := strings.ToLower(strings.TrimSpace(url))
	return strings.Contains(u, "spotify.com") || strings.HasPrefix(u, "spotify:")
}

// downloadYouTube downloads a YouTube URL using the pure-Go engine, falling
// back to the legacy pipeline if the engine errors out or produces nothing.
func downloadYouTube(url, outputDir string) *Result {
	if url == "" {
		return &Result{Status: "error", Error: "invalid URL"}
	}

	CurrentDownload.Set(true, 0, "Bridge: starting YouTube download...")

	if res, ok := runEngine(url, outputDir); ok {
		return res
	}

	filePath, err := music.DownloadYouTubeToFile(url, outputDir, func(pct int, msg string) {
		CurrentDownload.Set(true, float64(pct), fmt.Sprintf("YouTube: %s", msg))
	})
	if err != nil {
		CurrentDownload.Set(false, 0, fmt.Sprintf("Error: %v", err))
		return &Result{Status: "error", Error: fmt.Sprintf("YouTube download failed: %v", err)}
	}

	CurrentDownload.Set(false, 100, fmt.Sprintf("Saved: %s", filepath.Base(filePath)))

	meta := extractMetadata(filePath)
	if meta.Status == "error" {
		meta = &Result{
			Status:   "ok",
			Filename: filepath.Base(filePath),
			Title:    strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)),
			Artist:   "Unknown",
		}
	}
	// Register so the track shows up in the playlist (song_list.txt).
	registerSong(outputDir, filePath, meta)
	return meta
}

// runEngine is the unified Go-engine dispatcher and the head of the
// error fallback chain. It tries the pure-Go engine first, routing by URL
// type, and verifies a real audio file was produced before declaring
// success. When the engine is disabled by env, returns an error, or writes
// nothing, it returns (nil, false) so the caller falls back to the legacy
// pipeline.
func runEngine(url, outputDir string) (*Result, bool) {
	if engineDisabled() {
		return nil, false
	}

	before := time.Now()
	prior := listAudioFiles(outputDir)

	if isSpotifyURL(url) {
		CurrentDownload.Set(true, 0, "Bridge: starting Spotify engine...")
		err := engine.DownloadSpotify(engine.SpotifyOptions{
			OutputDir: outputDir,
			URLs:      []string{url},
			Progress:  engineProgress,
		})
		if err != nil {
			CurrentDownload.Set(false, 0, fmt.Sprintf("Engine error, trying legacy method: %v", err))
			return nil, false
		}
	} else {
		CurrentDownload.Set(true, 0, "Bridge: starting Go engine...")
		err := engine.DownloadYouTube(engine.YouTubeOptions{
			OutputDir: outputDir,
			URLs:      []string{url},
			Progress:  engineProgress,
		})
		if err != nil {
			CurrentDownload.Set(false, 0, fmt.Sprintf("Engine error, trying legacy method: %v", err))
			return nil, false
		}
	}

	// Fallback chain: engine returned fine but produced no file (e.g.
	// unknown URL, empty playlist) → drop to the legacy method.
	files := newAudioFiles(outputDir, before, prior)
	if len(files) == 0 {
		CurrentDownload.Set(false, 0, "Engine produced no file, trying legacy method")
		return nil, false
	}

	// Tek dosya → tek sonuç; çok dosya (çalma listesi) → şarkı listesi.
	if len(files) == 1 {
		res := resultFromFile(files[0])
		registerSong(outputDir, files[0], res)
		CurrentDownload.Set(false, 100, fmt.Sprintf("Completed via engine: %s", filepath.Base(files[0])))
		return res, true
	}

	songs := make([]Result, 0, len(files))
	for _, f := range files {
		res := resultFromFile(f)
		registerSong(outputDir, f, res)
		songs = append(songs, *res)
	}
	CurrentDownload.Set(false, 100, fmt.Sprintf("Downloaded %d song(s) via engine", len(songs)))
	return &Result{
		Status:  "ok",
		Message: fmt.Sprintf("Downloaded %d song(s)", len(songs)),
		Songs:   songs,
	}, true
}

// registerSong appends an engine-produced audio file to the playlist's
// song_list.txt so it shows up in the song list. Files already listed are
// skipped to avoid duplicate entries on re-downloads.
func registerSong(outputDir, filePath string, meta *Result) {
	base := filepath.Base(filePath)
	listPath := filepath.Join(outputDir, "song_list.txt")
	if existing, err := state.ReadSongs(listPath); err == nil {
		for _, s := range existing {
			if s.Filename == base {
				return
			}
		}
	}
	title := meta.Title
	if title == "" {
		title = strings.TrimSuffix(base, filepath.Ext(base))
	}
	artist := meta.Artist
	if artist == "" {
		artist = "Unknown"
	}
	_ = state.AppendSong(listPath, base, title, artist, fmtDuration(meta.Duration))
}

// engineProgress forwards the engine's progress events to the shared UI state.
func engineProgress(pct int, msg string) {
	CurrentDownload.Set(true, float64(pct), msg)
}

// audioExts are the container extensions the engines may emit.
var audioExts = map[string]bool{
	".mp3": true, ".m4a": true, ".webm": true, ".ogg": true, ".opus": true,
}

// resultFromFile builds a Result for an engine-produced file, enriching it
// with metadata when possible, else deriving a title from the filename.
func resultFromFile(filePath string) *Result {
	meta := extractMetadata(filePath)
	if meta.Status == "ok" && meta.Title != "" {
		return meta
	}
	base := filepath.Base(filePath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	return &Result{
		Status:   "ok",
		Filename: base,
		Title:    name,
		Artist:   "Unknown",
	}
}

// listAudioFiles returns the set of audio filenames currently in outputDir.
func listAudioFiles(outputDir string) map[string]struct{} {
	out := make(map[string]struct{})
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if audioExts[strings.ToLower(filepath.Ext(e.Name()))] {
			out[e.Name()] = struct{}{}
		}
	}
	return out
}

// newAudioFiles returns the paths of every audio file in outputDir that was
// created after `before` and was not present beforehand, sorted by modification
// time (oldest first, i.e. download order). An empty slice signals the engine
// produced nothing new, prompting the caller to fall back to the legacy
// pipeline. For playlists this yields all tracks; for a single track just one.
func newAudioFiles(outputDir string, before time.Time, prior map[string]struct{}) []string {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return nil
	}
	type hit struct {
		path string
		mod  time.Time
	}
	var hits []hit
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !audioExts[strings.ToLower(filepath.Ext(e.Name()))] {
			continue
		}
		if _, seen := prior[e.Name()]; seen {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		mod := info.ModTime()
		if mod.Before(before) {
			continue
		}
		hits = append(hits, hit{filepath.Join(outputDir, e.Name()), mod})
	}
	if len(hits) == 0 {
		return nil
	}
	// Insertion sort by mod time (small N); stable enough for display order.
	for i := 1; i < len(hits); i++ {
		for j := i; j > 0 && hits[j-1].mod.After(hits[j].mod); j-- {
			hits[j-1], hits[j] = hits[j], hits[j-1]
		}
	}
	out := make([]string, len(hits))
	for i, h := range hits {
		out[i] = h.path
	}
	return out
}

// downloadSpotify downloads a Spotify URL. Handles both track and playlist URLs.
func downloadSpotify(url, outputDir string) *Result {
	if url == "" {
		return &Result{Status: "error", Error: "invalid URL"}
	}

	CurrentDownload.Set(true, 0, "Starting...")

	urlLower := strings.ToLower(url)
	isPlaylist := strings.Contains(urlLower, "/playlist/") || strings.Contains(urlLower, "spotify:playlist:")
	isAlbum := strings.Contains(urlLower, "/album/") || strings.Contains(urlLower, "spotify:album:")
	isTrack := strings.Contains(urlLower, "/track/") || strings.Contains(urlLower, "spotify:track:")

	if isPlaylist || isAlbum {
		// Collections are handled by the legacy pipeline by design (the
		// engine resolves single tracks only). Skip the engine
		// attempt so a normal playlist path never flashes "Engine error".
		return downloadSpotifyPlaylist(url, outputDir)
	}

	if res, ok := runEngine(url, outputDir); ok {
		return res
	}

	if !isTrack {
		// Try collection (playlist/album) detection as fallback
		return downloadSpotifyPlaylist(url, outputDir)
	}

	// Single track: fetch Spotify metadata, search YouTube, download
	spTrack, err := music.FetchSpotifyTrack(url)
	if err != nil {
		CurrentDownload.Set(false, 0, "Error")
		return &Result{Status: "error", Error: fmt.Sprintf("Spotify metadata: %v", err)}
	}

	query := spTrack.Artist + " - " + spTrack.Title
	CurrentDownload.Set(true, 5, fmt.Sprintf("Searching YouTube: %s", query))

	videoID, _, err := music.SearchYouTubeTrack(query)
	if err != nil {
		CurrentDownload.Set(false, 0, "Error")
		return &Result{Status: "error", Error: fmt.Sprintf("YouTube search: %v", err)}
	}

	CurrentDownload.Set(true, 10, "Downloading from YouTube...")

	// Try direct MP3 download via yt-dlp + ffmpeg first (most reliable)
	ytTrack, mp3Data, err := music.DownloadYouTubeTrackDirectMP3(videoID, func(pct int, msg string) {
		CurrentDownload.Set(true, 10+float64(pct)*80/100, msg)
	})
	if err == nil && mp3Data != nil {
		CurrentDownload.Set(true, 90, "Writing ID3 tag...")

		spTrack.StreamURL = "https://www.youtube.com/watch?v=" + videoID
		spTrack.Format = "mp3"
		spTrack.ContentLen = int64(len(mp3Data))
		// Carry artwork so the cover gets embedded (search enrichment is
		// sometimes title-only, hence the hqdefault fallback).
		if spTrack.Thumbnail == "" {
			ytThumb := ""
			if ytTrack != nil {
				ytThumb = ytTrack.Thumbnail
			}
			spTrack.Thumbnail = music.VideoThumbnail(videoID, ytThumb)
		}

		tagged, err := music.SaveRawAsMP3(mp3Data, spTrack, outputDir, func(pct int, msg string) {
			CurrentDownload.Set(true, 90+float64(pct)*10/100, msg)
		})
		if err != nil {
			CurrentDownload.Set(false, 0, "Error")
			return &Result{Status: "error", Error: fmt.Sprintf("tag: %v", err)}
		}

		CurrentDownload.Set(false, 100, "Done")
		res := &Result{
			Status:   "ok",
			Filename: filepath.Base(tagged),
			Title:    spTrack.Title,
			Artist:   spTrack.Artist,
			Duration: spTrack.DurationSec,
		}
		registerSong(outputDir, tagged, res)
		return res
	}

	// Fallback: download raw WebM and convert
	CurrentDownload.Set(true, 10, "Direct MP3 failed, trying raw download...")
	ytTrack, rawAudio, err := music.DownloadYouTubeTrack(videoID, func(pct int, msg string) {
		CurrentDownload.Set(true, 10+float64(pct)*30/100, msg)
	})
	if err != nil {
		CurrentDownload.Set(false, 0, "Error")
		return &Result{Status: "error", Error: fmt.Sprintf("YouTube download: %v", err)}
	}

	CurrentDownload.Set(true, 40, "Converting to MP3...")

	spTrack.StreamURL = "https://www.youtube.com/watch?v=" + videoID
	spTrack.Format = "webm"
	spTrack.ContentLen = int64(len(rawAudio))
	if spTrack.Thumbnail == "" {
		ytThumb := ""
		if ytTrack != nil {
			ytThumb = ytTrack.Thumbnail
		}
		spTrack.Thumbnail = music.VideoThumbnail(videoID, ytThumb)
	}

	filePath, err := music.SaveRawAsMP3(rawAudio, spTrack, outputDir, func(pct int, msg string) {
		CurrentDownload.Set(true, 40+float64(pct)*60/100, msg)
	})
	if err != nil {
		CurrentDownload.Set(false, 0, "Error")
		return &Result{Status: "error", Error: fmt.Sprintf("convert: %v", err)}
	}

		CurrentDownload.Set(false, 100, "Done")

		res := &Result{
			Status:   "ok",
			Filename: filepath.Base(filePath),
			Title:    spTrack.Title,
			Artist:   spTrack.Artist,
			Duration: spTrack.DurationSec,
		}
		registerSong(outputDir, filePath, res)
		return res
	}

// downloadSpotifyPlaylist goes straight to the legacy API-free collection
// scraper. The engine resolves single tracks only, so attempting it
// here would only produce a bogus "Engine error" for a normal path.
func downloadSpotifyPlaylist(spotifyURL, outputDir string) *Result {
	files, err := playlist.DownloadSpotifyPlaylist(spotifyURL, outputDir, func(pct int, msg string) {
		CurrentDownload.Set(true, float64(pct), msg)
	})
	if err != nil {
		CurrentDownload.Set(false, 0, "Error")
		return &Result{Status: "error", Error: err.Error()}
	}

	CurrentDownload.Set(false, 100, fmt.Sprintf("Downloaded %d songs", len(files)))

	songs := make([]Result, 0, len(files))
	for _, f := range files {
		meta := extractMetadata(f)
		if meta.Status == "ok" {
			songs = append(songs, *meta)
		} else {
			meta = resultFromFile(f)
		}
		// Register every downloaded file so tracks show up in the
		// playlist (song_list.txt). registerSong skips duplicates.
		registerSong(outputDir, f, meta)
	}

	return &Result{
		Status:  "ok",
		Message: fmt.Sprintf("Downloaded %d song(s)", len(songs)),
		Songs:   songs,
	}
}
