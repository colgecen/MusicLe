package audio

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"MusicLe/internal/td/model"
	"MusicLe/internal/td/util"
	"github.com/dhowden/tag"
)

// youtubeIDSuffix matches the " [videoid]" suffix added by the downloader.
var youtubeIDSuffix = regexp.MustCompile(`\s*\[[A-Za-z0-9_-]{11}\]$`)

// ExtractMetadata reads audio file tags and returns AudioFileMetadata.
func ExtractMetadata(filePath string) *model.AudioFileMetadata {
	base := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	base = youtubeIDSuffix.ReplaceAllString(base, "")
	ext := strings.ToLower(filepath.Ext(filePath))
	result := &model.AudioFileMetadata{
		Status:   "ok",
		Filename: filepath.Base(filePath),
		Title:    base,
		Artist:   "Unknown",
		Album:    "",
		Format:   util.FormatFromExt(ext),
		Duration: 0,
	}

	f, err := os.Open(filePath)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("open file: %v", err)
		return result
	}
	defer f.Close()

	meta, err := tag.ReadFrom(f)
	if err == nil {
		if title := meta.Title(); title != "" {
			result.Title = title
		}
		if artist := meta.Artist(); artist != "" {
			result.Artist = artist
		}
		if album := meta.Album(); album != "" {
			result.Album = album
		}
	}

	// Artist fallback: parse "Artist - Title" from filename when artist unknown.
	if result.Artist == "Unknown" || result.Artist == "" {
		if strings.Contains(base, " - ") {
			parts := strings.SplitN(base, " - ", 2)
			if strings.TrimSpace(parts[0]) != "" {
				result.Artist = strings.TrimSpace(parts[0])
			}
			// Only override title if it still equals the full base name.
			if result.Title == base && len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
				result.Title = strings.TrimSpace(parts[1])
			}
		}
	}
	if result.Artist == "" {
		result.Artist = "Unknown"
	}
	if result.Title == "" {
		result.Title = base
	}

	// Duration only for MP3.
	if ext == ".mp3" {
		if secs, err := MP3DurationSec(filePath); err == nil && secs > 0 {
			result.Duration = int(secs)
		}
	}

	return result
}

// ReadCover reads the embedded picture from an audio file.
func ReadCover(filePath string) ([]byte, string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	meta, err := tag.ReadFrom(f)
	if err != nil {
		return nil, "", err
	}
	pic := meta.Picture()
	if pic == nil {
		return nil, "", fmt.Errorf("no embedded picture")
	}
	mime := pic.MIMEType
	if mime == "" {
		mime = "image/jpeg"
	}
	return pic.Data, mime, nil
}
