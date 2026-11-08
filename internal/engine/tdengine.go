package engine

import (
	"context"
	"fmt"
	"os"

	tddl "MusicLe/internal/td/download"
	tdyt "MusicLe/internal/td/youtube"
)

// noopProgress discards progress events; used when the caller passes none.
func noopProgress(int, string) {}

// cookieEnvVar points the engine at a Netscape cookies.txt the user exported
// from a signed-in browser session. It is opt-in: without it the engine stays
// fully anonymous, which is what most videos need anyway.
const cookieEnvVar = "MUSICLE_COOKIES"

// applyCookieFile wires MUSICLE_COOKIES into the YouTube client when set.
// A bad path is reported instead of silently downloading without cookies,
// because the user explicitly asked for them.
func applyCookieFile() error {
	path := os.Getenv(cookieEnvVar)
	if path == "" {
		return nil
	}
	return tdyt.SetCookieFile(path)
}

// downloadWithTD is the pure-Go download pipeline.
//
// When strict is set, every input must already be a YouTube link or id (the
// DownloadYouTube entry point); anything else fails instead of turning into a
// search, matching what callers and tests expect. When strict is clear, free
// text is searched on YouTube and the best-scoring result is downloaded — that
// is how Spotify titles reach YouTube.
//
// Playlist references are expanded into their videos and downloaded through
// the same pipeline. Progress for input i of n is mapped onto the
// [i·100/n, (i+1)·100/n] slice so the UI bar advances once per input.
func downloadWithTD(urls []string, outDir string, strict bool, progress Progress) error {
	if progress == nil {
		progress = noopProgress
	}
	if len(urls) == 0 {
		return fmt.Errorf("engine: en az bir URL gerekli")
	}
	if err := applyCookieFile(); err != nil {
		return fmt.Errorf("engine: cookies dosyasi okunamadi: %w", err)
	}
	// MusicLe always hands the player MP3; the engine converts whenever
	// YouTube served Opus/AAC instead.
	tddl.SetMP3Output(true)

	n := len(urls)
	for i, u := range urls {
		lo, hi := i*100/n, (i+1)*100/n
		progress(lo, fmt.Sprintf("Downloading %d/%d...", i+1, n))
		target := tdyt.Classify(u)
		switch {
		case target.Kind == tdyt.KindPlaylist:
			if err := downloadPlaylistGo(u, outDir, func(pct int, msg string) {
				progress(lo+pct*(hi-lo)/100, msg)
			}); err != nil {
				return err
			}
		case target.Kind == tdyt.KindVideo || !strict:
			if _, err := downloadOneGo(u, outDir, func(pct int, msg string) {
				progress(lo+pct*(hi-lo)/100, msg)
			}); err != nil {
				return err
			}
		default:
			return fmt.Errorf("engine: %q gecerli bir YouTube baglantisi veya video kimligi degil", u)
		}
	}
	progress(100, "Done")
	return nil
}

// downloadOneGo downloads and converts a single input, reporting phase-aware
// progress (transfer first, MP3 encoding from tddl.EncodeProgressStart on).
func downloadOneGo(input, outDir string, progress Progress) (string, error) {
	_, filePath, err := tddl.DownloadYouTube(context.Background(), input, outDir, "", 0, func(pct float64) {
		phase := "Downloading audio..."
		if pct >= tddl.EncodeProgressStart {
			phase = "Converting to MP3..."
		}
		progress(int(pct), fmt.Sprintf("%s %d%%", phase, int(pct)))
	})
	if err != nil {
		return "", err
	}
	return filePath, nil
}

// downloadPlaylistGo expands a playlist URL and downloads every playable
// video, reporting per-track progress inside the parent input's slice.
func downloadPlaylistGo(playlistURL, outDir string, progress Progress) error {
	items, _, err := tddl.DownloadYouTubePlaylist(context.Background(), playlistURL, outDir, "", 2,
		func(idx, total int, title string, pct float64) {
			if total < 1 {
				total = 1
			}
			overall := int((float64(idx-1) + pct/100) * 100 / float64(total))
			progress(overall, fmt.Sprintf("%d/%d %s... %d%%", idx, total, title, int(pct)))
		})
	if err != nil {
		return err
	}
	failed := 0
	for _, it := range items {
		if it.Err != nil {
			failed++
		}
	}
	if failed == len(items) && len(items) > 0 {
		return fmt.Errorf("engine: calisma listesindeki hicbir video indirilemedi")
	}
	return nil
}
