package engine

import (
	"fmt"
	"strings"
)

// DownloadYouTube downloads one or more YouTube URLs to MP3 with an embedded
// cover using the pure-Go TerminalDownloader engine. It returns the first
// non-fatal error if any URL failed, nil when every URL was processed.
func DownloadYouTube(opt YouTubeOptions) error {
	urls := normalizeYouTubeURLs(opt.URLs)
	if len(urls) == 0 {
		return fmt.Errorf("engine: en az bir YouTube URL gerekli")
	}
	return downloadWithTD(urls, opt.OutputDir, true, opt.Progress)
}

// DownloadYouTubePlaylist is a thin alias for DownloadYouTube with a single
// URL — the engine resolves playlist URLs through the bridge-side playlist
// downloader, which fans the list out into per-track calls.
func DownloadYouTubePlaylist(opt YouTubeOptions) error {
	return DownloadYouTube(opt)
}

// normalizeYouTubeURLs trims whitespace and drops empty entries; the engine
// is tolerant of any URL shape (watch?v=, youtu.be/, bare IDs, search text),
// so we don't rewrite the URL itself.
func normalizeYouTubeURLs(in []string) []string {
	out := make([]string, 0, len(in))
	for _, raw := range in {
		u := strings.TrimSpace(raw)
		if u == "" {
			continue
		}
		out = append(out, u)
	}
	return out
}
