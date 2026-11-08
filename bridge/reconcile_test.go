package bridge

import (
	"os"
	"path/filepath"
	"testing"

	"MusicLe/state"
)

func withTempLibrary(t *testing.T) string {
	t.Helper()
	saved := state.Current
	t.Cleanup(func() { state.Current = saved })
	root := t.TempDir()
	state.Current = &state.AppState{RootDir: root}
	plDir := filepath.Join(root, "profiles", "u", "playlists", "pl")
	if err := os.MkdirAll(plDir, 0755); err != nil {
		t.Fatal(err)
	}
	return plDir
}

// Orphan mp3 files must be registered into song_list.txt.
func TestReconcileLibraryAddsOrphans(t *testing.T) {
	plDir := withTempLibrary(t)
	for _, f := range []string{"A - One.mp3", "B - Two.mp3", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(plDir, f), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	listPath := filepath.Join(plDir, "song_list.txt")
	if err := state.AppendSong(listPath, "A - One.mp3", "One", "A", "03:00"); err != nil {
		t.Fatal(err)
	}
	if err := state.Current.ScanProfiles(); err != nil {
		t.Fatal(err)
	}
	if got := ReconcileLibrary(); got != 1 {
		t.Fatalf("1 yetim bekleniyordu, %d eklendi", got)
	}
	songs, err := state.ReadSongs(listPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(songs) != 2 {
		t.Fatalf("2 kayit bekleniyordu, %d var", len(songs))
	}
	if songs[1].Artist != "B" || songs[1].Title != "Two" {
		t.Errorf("yanlis cozumleme: %+v", songs[1])
	}
	// Second run adds nothing (idempotent).
	if got := ReconcileLibrary(); got != 0 {
		t.Errorf("ikinci calisma 0 eklemeliydi, %d ekledi", got)
	}
}
