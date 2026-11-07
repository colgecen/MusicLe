package util

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// SupportedAudioExt maps file extension -> canonical format name.
var SupportedAudioExt = map[string]string{
	".mp3":  "MP3",
	".flac": "FLAC",
	".m4a":  "AAC",
	".mp4":  "AAC",
	".aac":  "AAC",
	".ogg":  "OGG",
	".wav":  "WAV",
	".opus": "Opus",
	// YouTube audio-only streams are Opus inside a WebM container.
	".webm": "Opus",
}

// AllowedAudioExt lists extensions accepted for local import.
var AllowedAudioExt = map[string]bool{
	".mp3": true,
}

// windowsReserved are device names that cannot be used as a file or directory
// stem on Windows, with or without an extension.
var windowsReserved = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
	"COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
	"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// SanitizeName replaces filesystem-unsafe characters and applies the Windows
// naming rules too, so a library created on Linux stays usable when copied to
// Windows: no trailing dots/spaces and no reserved device names.
func SanitizeName(name string) string {
	name = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	// Windows strips trailing dots and spaces silently, which would make the
	// name on disk differ from the name in song_list.txt.
	name = strings.TrimRight(name, " .")
	if name == "" {
		return ""
	}
	stem := name
	if i := strings.IndexByte(stem, '.'); i >= 0 {
		stem = stem[:i]
	}
	if windowsReserved[strings.ToUpper(stem)] {
		name = "_" + name
	}
	return name
}

// FmtDuration formats seconds as mm:ss.
func FmtDuration(seconds float64) string {
	s := int(seconds)
	return fmt.Sprintf("%02d:%02d", s/60, s%60)
}

// NormalizeDuration ensures a duration string is mm:ss.
func NormalizeDuration(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "00:00"
	}
	if strings.Contains(s, ":") {
		return s
	}
	secs, err := strconv.Atoi(s)
	if err != nil {
		return s
	}
	return fmt.Sprintf("%02d:%02d", secs/60, secs%60)
}

// FormatFromExt returns canonical format from extension.
func FormatFromExt(ext string) string {
	return SupportedAudioExt[strings.ToLower(ext)]
}

// IsAudioFile reports whether path has a supported audio extension.
func IsAudioFile(path string) bool {
	_, ok := SupportedAudioExt[strings.ToLower(filepath.Ext(path))]
	return ok
}

// Today returns current date as YYYY-MM-DD.
func Today() string {
	return time.Now().Format("2006-01-02")
}

// NowDateTime returns current local time as YYYY-MM-DD HH:MM.
func NowDateTime() string {
	return time.Now().Format("2006-01-02 15:04")
}

// ReadTxt reads a text file, returning def on error.
func ReadTxt(path, def string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return def
	}
	return strings.TrimSpace(string(data))
}

// WriteTxt writes content to path, creating parent dirs.
func WriteTxt(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// CopyFile copies src to dst.
func CopyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
