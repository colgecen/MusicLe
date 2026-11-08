package bridge

import (
	"os"
	"path/filepath"
	"strings"

	"MusicLe/state"
)

// ReconcileLibrary scans every playlist directory and registers orphan audio
// files (on disk but missing from song_list.txt). Downloads that finished
// without registration (crash, old version, manual copies) show up in the UI
// again after this runs. It is idempotent — call it at startup and after
// downloads. Returns the number of entries added.
func ReconcileLibrary() int {
	added := 0
	for _, p := range state.Current.Profiles {
		for _, pl := range p.Playlists {
			added += reconcileDir(state.Current.PlaylistDir(p.FolderName, pl.FolderName))
		}
	}
	return added
}

func reconcileDir(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	listPath := filepath.Join(dir, "song_list.txt")
	existing, _ := state.ReadSongs(listPath)
	have := make(map[string]bool, len(existing))
	for _, s := range existing {
		have[s.Filename] = true
	}
	// Backfill missing durations of already-listed songs (older repairs
	// wrote "00:00").
	for _, s := range existing {
		if s.Duration == "" || s.Duration == "00:00" {
			full := filepath.Join(dir, s.Filename)
			if secs, err := MP3DurationSec(full); err == nil && secs > 0 {
				_ = state.UpdateSong(listPath, s.Filename, "", "", fmtDuration(secs))
			}
		}
	}
	added := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !audioExts[strings.ToLower(filepath.Ext(e.Name()))] {
			continue
		}
		if have[e.Name()] {
			continue
		}
		full := filepath.Join(dir, e.Name())
		meta := extractMetadata(full)
		// Downloads carry ID3 tags, but be lenient: split
		// "Artist - Title.mp3" filenames when tags are absent.
		if meta.Artist == "" || meta.Artist == "Unknown" {
			if a, t := splitArtistTitle(e.Name()); a != "" && t != "" {
				meta.Artist, meta.Title = a, t
			}
		}
		registerSong(dir, full, meta)
		added++
	}
	return added
}

// CleanUnusedArtwork removes leftover artwork folders that the app no
// longer uses: per-playlist playlist_avatar/ dirs and _art/ sidecar dirs.
// Covers now come from embedded audio pictures, so these are dead weight.
// Returns the number of removed directories.
func CleanUnusedArtwork() int {
	removed := 0
	for _, p := range state.Current.Profiles {
		for _, pl := range p.Playlists {
			dir := state.Current.PlaylistDir(p.FolderName, pl.FolderName)
			for _, stale := range []string{"playlist_avatar", "_art"} {
				full := filepath.Join(dir, stale)
				if fi, err := os.Stat(full); err == nil && fi.IsDir() {
					if err := os.RemoveAll(full); err == nil {
						removed++
					}
				}
			}
		}
		// Legacy per-profile avatar folder (option removed).
		avatarDir := filepath.Join(state.Current.ProfilesDir(), p.FolderName, "avatar")
		if fi, err := os.Stat(avatarDir); err == nil && fi.IsDir() {
			if err := os.RemoveAll(avatarDir); err == nil {
				removed++
			}
		}
	}
	return removed
}

// splitArtistTitle splits "Artist - Title.mp3" into its parts.
func splitArtistTitle(filename string) (artist, title string) {
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	parts := strings.SplitN(base, " - ", 2)
	if len(parts) != 2 {
		return "", strings.TrimSpace(base)
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}
