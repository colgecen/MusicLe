package bridge

import (
	"os"
	"path/filepath"
	"testing"

	"MusicLe/state"
)

// TestRegisterSongWritesSongList guards the "downloads but not in playlist"
// bug: every successful download path must append to song_list.txt, otherwise
// the UI (which reads that file) never shows the track.
func TestRegisterSongWritesSongList(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "Artist - Title.mp3")
	if err := os.WriteFile(f, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	meta := &Result{Status: "ok", Filename: "Artist - Title.mp3", Title: "Title", Artist: "Artist", Duration: 234}
	registerSong(dir, f, meta)

	songs, err := state.ReadSongs(filepath.Join(dir, "song_list.txt"))
	if err != nil {
		t.Fatalf("song_list.txt yazilmamis: %v", err)
	}
	if len(songs) != 1 {
		t.Fatalf("1 sarki bekleniyordu, %d var", len(songs))
	}
	if songs[0].Title != "Title" || songs[0].Artist != "Artist" {
		t.Errorf("yanlis kayit: %+v", songs[0])
	}

	// Second registration of the same file must not duplicate.
	registerSong(dir, f, meta)
	songs, err = state.ReadSongs(filepath.Join(dir, "song_list.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(songs) != 1 {
		t.Errorf("tekrar kayit olustu: %d satir", len(songs))
	}
}
