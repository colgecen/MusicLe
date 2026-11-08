package download

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"MusicLe/internal/td/encode"
	"MusicLe/internal/td/model"
)

// mp3Output makes every completed download end up as an MP3. It is process-wide
// because the CLI is a single-shot tool and every download path must honour it.
var mp3Output bool

// SetMP3Output enables or disables MP3 conversion for subsequent downloads.
func SetMP3Output(on bool) { mp3Output = on }

// MP3Output reports whether downloads are converted to MP3.
func MP3Output() bool { return mp3Output }

// maybeConvertToMP3 turns a finished WebM/Opus or MP4/AAC download into MP3
// when the feature is enabled. It returns the path that should be recorded in
// the playlist; on failure the original file is kept and the error is returned.
func maybeConvertToMP3(ctx context.Context, ti *model.TrackInfo, filePath string, progress func(float64)) (string, error) {
	if !mp3Output || strings.EqualFold(filepath.Ext(filePath), ".mp3") {
		return filePath, nil
	}
	if !encode.IsSupportedSource(filePath) {
		return filePath, encode.ErrUnsupportedSource
	}

	dst := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".mp3"
	tags := encode.Tags{
		Title:  tagTitle(ti.Artist, ti.Title),
		Artist: ti.Artist,
		Album:  ti.Album,
	}
	if ti.Thumbnail != "" {
		if data, mime, err := fetchImage(ctx, ti.Thumbnail); err == nil {
			tags.Cover, tags.CoverMIME = data, mime
		}
	}

	// Encoding is the second half of the overall progress bar, which the
	// download phase filled up to EncodeProgressStart.
	if err := encode.ToMP3(ctx, filePath, dst, float64(ti.DurationSec), tags, scaleProgress(progress, EncodeProgressStart, 100)); err != nil {
		return filePath, fmt.Errorf("mp3 donusumu: %w", err)
	}
	ti.Format = "mp3"
	if st, err := os.Stat(dst); err == nil {
		ti.ContentLen = st.Size()
	}
	// The MP3 is complete even if the source cannot be removed (permissions,
	// read-only mount); do not report the conversion as failed in that case.
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "  uyari: kaynak dosya silinemedi: %v\n", err)
	}
	return dst, nil
}

// EncodeProgressStart is where download progress stops and MP3 encoding
// starts, so the bar moves once instead of resetting between the two phases.
// Hosts (like the MusicLe engine) use it to label the current phase.
const EncodeProgressStart = 70.0

// scaleProgress maps a 0-100 callback onto the [lo, hi] slice of a single
// progress bar. A nil callback stays nil.
func scaleProgress(p func(float64), lo, hi float64) func(float64) {
	if p == nil {
		return nil
	}
	return func(v float64) {
		if v < 0 {
			v = 0
		}
		if v > 100 {
			v = 100
		}
		p(lo + (hi-lo)*v/100)
	}
}

// tagTitle drops a leading "Artist - " from the title so the ID3 title frame
// does not repeat the artist frame.
func tagTitle(artist, title string) string {
	title = strings.TrimSpace(title)
	artist = strings.TrimSpace(artist)
	if artist == "" || !strings.HasPrefix(strings.ToLower(title), strings.ToLower(artist)) {
		return title
	}
	rest := strings.TrimSpace(title[len(artist):])
	rest = strings.TrimLeft(rest, "-–—:|,")
	return strings.TrimSpace(rest)
}

// fetchImage downloads a small image (cover art) and returns its bytes and MIME
// type. Failures are non-fatal for the caller.
func fetchImage(ctx context.Context, url string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("gorsel status %d", resp.StatusCode)
	}
	// Cover art never needs to be huge; cap it so a bad URL cannot eat memory.
	data, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	if err != nil {
		return nil, "", err
	}
	mime := resp.Header.Get("Content-Type")
	if i := strings.Index(mime, ";"); i >= 0 {
		mime = strings.TrimSpace(mime[:i])
	}
	if mime == "" || mime == "application/octet-stream" {
		mime = "image/jpeg"
	}
	return data, mime, nil
}
