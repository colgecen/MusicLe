package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// DownloadSpotify downloads Spotify tracks without any API credentials. Each
// track is resolved through Spotify's public embed page (no token) to its
// artist + title, then downloaded from YouTube with the pure-Go engine — the
// engine scores the search results instead of blindly taking the first hit —
// and muxed to MP3 with an embedded cover, the same pipeline as
// DownloadYouTube.
//
// Playlist / album URLs are not resolved here (embed pages do not expose the
// full track list without an API); they return an error so the bridge falls
// back to its legacy API-free collection scraper.
func DownloadSpotify(opt SpotifyOptions) error {
	inputs := sanitizeSpotifyInputs(opt.URLs)
	if len(inputs) == 0 {
		return fmt.Errorf("spotify: at least one URL or search query is required")
	}
	for _, in := range inputs {
		query, err := spotifyQueryFor(in)
		if err != nil {
			return err
		}
		if err := downloadWithTD([]string{query}, opt.OutputDir, false, opt.Progress); err != nil {
			return err
		}
	}
	return nil
}

// DownloadSpotifyPlaylist is kept as a separate entry point for playlist /
// album flows. Spotify collections need the API-free collection scraper on
// the bridge side, so this reports an error here only when a collection URL
// slips through.
func DownloadSpotifyPlaylist(opt SpotifyOptions) error {
	return DownloadSpotify(opt)
}

func sanitizeSpotifyInputs(in []string) []string {
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

// spotifyEmbedRe matches the [type]/[id] tail of a Spotify URL, and the
// spotify:type:id form.
var spotifyEmbedRe = regexp.MustCompile(`(?:open\.spotify\.com/|spotify:)(track|album|playlist)(?:/|:)([A-Za-z0-9]+)`)

// spotifyQueryFor turns a Spotify track URL into a YouTube search query, or a
// plain non-Spotify string into itself (so raw "artist - title" searches are
// supported too).
func spotifyQueryFor(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if !strings.Contains(trimmed, "spotify.") && !strings.HasPrefix(trimmed, "spotify:") {
		if trimmed == "" {
			return "", fmt.Errorf("spotify: empty query")
		}
		return trimmed, nil
	}

	m := spotifyEmbedRe.FindStringSubmatch(trimmed)
	if m == nil {
		return "", fmt.Errorf("spotify: unsupported URL format: %s", trimmed)
	}
	ent, id := m[1], m[2]
	if ent != "track" {
		return "", fmt.Errorf("spotify: page %s is not supported via scrape in this version; falling back to bridge collector", ent)
	}

	name, artist, err := scrapeSpotifyTrack(id)
	if err != nil {
		return "", err
	}
	if artist != "" {
		return artist + " - " + name, nil
	}
	return name, nil
}

type spotifyEmbedEntity struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Artists []struct {
		Name string `json:"name"`
	} `json:"artists"`
}

type spotifyEmbedState struct {
	Props *struct {
		PageProps *struct {
			State *struct {
				Data *struct {
					Entity *spotifyEmbedEntity `json:"entity"`
				} `json:"data"`
			} `json:"state"`
		} `json:"pageProps"`
	} `json:"props"`
}

var nextDataRe = regexp.MustCompile(`<script[^>]*id=["']__NEXT_DATA__["'][^>]*>(.*?)</script>`)

// scrapeSpotifyTrack fetches a track's artist + title from Spotify's public
// embed page. No API key, no credentials — exactly like a browser requesting
// the same widget a user embeds on a web page.
func scrapeSpotifyTrack(id string) (name, artist string, err error) {
	pageURL := "https://open.spotify.com/embed/track/" + url.PathEscape(id)

	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("spotify: cannot fetch page: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("spotify: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", "", err
	}

	if m := nextDataRe.FindSubmatch(body); m != nil {
		var st spotifyEmbedState
		if json.Unmarshal(m[1], &st) == nil {
			if e := st.Props.PageProps.State.Data.Entity; e != nil && e.Name != "" {
				if len(e.Artists) > 0 {
					artist = e.Artists[0].Name
				}
				return unescapeHTML(e.Name), unescapeHTML(artist), nil
			}
		}
	}
	return "", "", fmt.Errorf("spotify: cannot extract track metadata (%s)", id)
}

func unescapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	return s
}
