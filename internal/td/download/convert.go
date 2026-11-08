package download

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"MusicLe/internal/td/audio"
	"MusicLe/internal/td/encode"
)

// ConvertToMP3 converts an existing WebM/Opus or MP4/AAC file into MP3 in
// place and removes the original. It reads whatever tags the source already has
// (or derives them from the file name) so the MP3 keeps its title/artist.
func ConvertToMP3(ctx context.Context, src string, progress func(float64)) (string, error) {
	ext := strings.ToLower(filepath.Ext(src))
	if ext == ".mp3" {
		return src, nil
	}
	if !encode.IsSupportedSource(src) {
		return src, encode.ErrUnsupportedSource
	}

	meta := audio.ExtractMetadata(src)
	tags := encode.Tags{Title: meta.Title, Artist: meta.Artist, Album: meta.Album}
	if cover, mime, err := audio.ReadCover(src); err == nil {
		tags.Cover, tags.CoverMIME = cover, mime
	}
	// YouTube's WebM files carry no embedded cover; fall back to the playlist
	// art next to the file so the MP3 is not left blank.
	if len(tags.Cover) == 0 {
		if artPath := findPlaylistArt(filepath.Dir(src)); artPath != "" {
			if data, err := os.ReadFile(artPath); err == nil {
				tags.Cover = data
				tags.CoverMIME = mimeForExt(artPath)
			}
		}
	}

	dst := strings.TrimSuffix(src, filepath.Ext(src)) + ".mp3"
	if err := encode.ToMP3(ctx, src, dst, float64(meta.Duration), tags, progress); err != nil {
		return src, err
	}
	if err := os.Remove(src); err != nil && !os.IsNotExist(err) {
		return dst, fmt.Errorf("eski dosya silinemedi: %w", err)
	}
	return dst, nil
}

// findPlaylistArt returns the cover image of a playlist folder, if present.
func findPlaylistArt(dir string) string {
	for _, name := range []string{"art.jpg", "art.jpeg", "art.png"} {
		p := filepath.Join(dir, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func mimeForExt(path string) string {
	if strings.EqualFold(filepath.Ext(path), ".png") {
		return "image/png"
	}
	return "image/jpeg"
}

// IsConvertible reports whether a file can be turned into MP3 by ConvertToMP3.
func IsConvertible(path string) bool {
	out, err := ConvertibleReason(path)
	return err == nil && out
}

// ConvertibleReason explains whether and why a file can be converted.
func ConvertibleReason(path string) (bool, error) {
	if strings.EqualFold(filepath.Ext(path), ".mp3") {
		return false, nil
	}
	if !encode.IsSupportedSource(path) {
		return false, encode.ErrUnsupportedSource
	}
	return true, nil
}
