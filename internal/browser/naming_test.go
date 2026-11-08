package browser

import (
	"os"
	"path/filepath"
	"testing"

	"MusicLe/state"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Spotify Profili":  "spotify_profili",
		"My Playlist - 2":  "my_playlist_2",
		"  Hello World  ":  "hello_world",
		"Türkçe İşletme?!": "trke_iletme",
		"":                 "playlist",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNextProfileName_First(t *testing.T) {
	dir := t.TempDir()
	name, folder := NextProfileName(nil, PlatformSpotify, dir)
	if name != "Spotify Profili" {
		t.Errorf("name = %q, want %q", name, "Spotify Profili")
	}
	if folder != "spotify_profili" {
		t.Errorf("folder = %q, want %q", folder, "spotify_profili")
	}
}

func TestNextProfileName_Increment(t *testing.T) {
	dir := t.TempDir()
	profiles := []state.Profile{{DisplayName: "Spotify Profili"}}
	name, _ := NextProfileName(profiles, PlatformSpotify, dir)
	if name != "Spotify Profili 2" {
		t.Errorf("name = %q, want %q", name, "Spotify Profili 2")
	}

	yt := []state.Profile{
		{DisplayName: "YouTube Profili"},
		{DisplayName: "YouTube Profili 2"},
	}
	yname, _ := NextProfileName(yt, PlatformYouTube, dir)
	if yname != "YouTube Profili 3" {
		t.Errorf("yname = %q, want %q", yname, "YouTube Profili 3")
	}
}

func TestNextProfileName_UniqueFolder(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "spotify_profili"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, folder := NextProfileName(nil, PlatformSpotify, dir)
	if folder != "spotify_profili_2" {
		t.Errorf("folder = %q, want %q", folder, "spotify_profili_2")
	}
}
