package audio

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func copyFixture(t *testing.T, name string) string {
	t.Helper()
	src, err := os.Open(filepath.Join("..", "testdata", name))
	if err != nil {
		t.Skipf("fixture yok: %v", err)
	}
	defer src.Close()
	dst := filepath.Join(t.TempDir(), name)
	out, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(out, src); err != nil {
		out.Close()
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
	return dst
}

func TestWriteMP3Tags(t *testing.T) {
	path := copyFixture(t, "sample.mp3")
	err := WriteMP3Tags(path, TagUpdate{Title: "Yeni Baslik", Artist: "Yeni Sanatci", Album: "Yeni Album"}, nil, "")
	if err != nil {
		t.Fatalf("WriteMP3Tags: %v", err)
	}
	meta := ExtractMetadata(path)
	if meta.Title != "Yeni Baslik" || meta.Artist != "Yeni Sanatci" || meta.Album != "Yeni Album" {
		t.Errorf("metadata = %+v", meta)
	}
}

func TestWriteMP3TagsAddsCover(t *testing.T) {
	source := filepath.Join("..", "testdata", "with_cover.mp3")
	cover, mime, err := ReadCover(source)
	if err != nil {
		t.Skipf("kapak fixture yok: %v", err)
	}
	path := copyFixture(t, "sample.mp3")
	if err := WriteMP3Tags(path, TagUpdate{Title: "Kapakli"}, cover, mime); err != nil {
		t.Fatalf("WriteMP3Tags: %v", err)
	}
	got, gotMIME, err := ReadCover(path)
	if err != nil {
		t.Fatalf("ReadCover: %v", err)
	}
	if len(got) != len(cover) {
		t.Errorf("cover size = %d, want %d", len(got), len(cover))
	}
	if gotMIME == "" {
		t.Error("empty cover MIME")
	}
}

func TestSplitArtistTitle(t *testing.T) {
	cases := []struct {
		name   string
		artist string
		title  string
	}{
		{"Rick Astley - Never Gonna Give You Up [dQw4w9WgXcQ].mp3", "Rick Astley", "Never Gonna Give You Up"},
		{"No Artist Here.mp3", "", "No Artist Here"},
		{"A - B - C.mp3", "A", "B - C"},
	}
	for _, c := range cases {
		artist, title := SplitArtistTitle(c.name)
		if artist != c.artist || title != c.title {
			t.Errorf("SplitArtistTitle(%q) = (%q,%q), want (%q,%q)", c.name, artist, title, c.artist, c.title)
		}
	}
}
