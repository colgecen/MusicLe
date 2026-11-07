package audio

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractMetadataFixtures(t *testing.T) {
	cases := []struct {
		file   string
		title  string
		artist string
		album  string
		format string
		dur    int
	}{
		{"sample.mp3", "Test Song", "Test Artist", "Test Album", "MP3", 2},
		{"sample.flac", "Flac Song", "Flac Artist", "", "FLAC", 0},
		{"sample.m4a", "AAC Song", "AAC Artist", "", "AAC", 0},
		{"sample.ogg", "Ogg Song", "Ogg Artist", "", "OGG", 0},
		{"sample.opus", "Opus Song", "Opus Artist", "", "Opus", 0},
		{"sample.wav", "sample", "Unknown", "", "WAV", 0},
	}
	for _, c := range cases {
		m := ExtractMetadata(filepath.Join("..", "testdata", c.file))
		if m.Status != "ok" {
			t.Fatalf("%s status = %s (%s)", c.file, m.Status, m.Error)
		}
		if m.Title != c.title {
			t.Errorf("%s title = %q, want %q", c.file, m.Title, c.title)
		}
		if m.Artist != c.artist {
			t.Errorf("%s artist = %q, want %q", c.file, m.Artist, c.artist)
		}
		if m.Album != c.album {
			t.Errorf("%s album = %q, want %q", c.file, m.Album, c.album)
		}
		if m.Format != c.format {
			t.Errorf("%s format = %q, want %q", c.file, m.Format, c.format)
		}
		if m.Duration != c.dur {
			t.Errorf("%s duration = %d, want %d", c.file, m.Duration, c.dur)
		}
	}
}

func TestExtractMetadataFilenameFallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Some Artist - Some Title [dQw4w9WgXcQ].mp3")
	if err := os.WriteFile(path, []byte("not audio"), 0644); err != nil {
		t.Fatal(err)
	}
	m := ExtractMetadata(path)
	if m.Status != "ok" {
		t.Fatalf("status = %s", m.Status)
	}
	if m.Title != "Some Title" {
		t.Errorf("title = %q, want %q", m.Title, "Some Title")
	}
	if m.Artist != "Some Artist" {
		t.Errorf("artist = %q, want %q", m.Artist, "Some Artist")
	}
	if m.Format != "MP3" {
		t.Errorf("format = %q", m.Format)
	}
}

func TestExtractMetadataMissingFile(t *testing.T) {
	m := ExtractMetadata(filepath.Join(t.TempDir(), "nope.mp3"))
	if m.Status != "error" {
		t.Fatalf("status = %s, want error", m.Status)
	}
	if m.Error == "" {
		t.Error("expected an error message")
	}
}

func TestMP3DurationSec(t *testing.T) {
	secs, err := MP3DurationSec(filepath.Join("..", "testdata", "sample.mp3"))
	if err != nil {
		t.Fatal(err)
	}
	if secs < 1.5 || secs > 2.5 {
		t.Errorf("duration = %v, want ~2", secs)
	}
}

func TestReadCoverNone(t *testing.T) {
	if _, _, err := ReadCover(filepath.Join("..", "testdata", "sample.mp3")); err == nil {
		t.Error("expected error for file without embedded cover")
	}
}

func TestReadCoverEmbedded(t *testing.T) {
	data, mime, err := ReadCover(filepath.Join("..", "testdata", "with_cover.mp3"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("empty cover")
	}
	if mime == "" {
		t.Error("empty mime")
	}
	if m := ExtractMetadata(filepath.Join("..", "testdata", "with_cover.mp3")); m.Title != "Covered Song" || m.Artist != "Covered Artist" {
		t.Errorf("metadata = %+v", m)
	}
}
