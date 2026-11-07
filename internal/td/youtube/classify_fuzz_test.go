package youtube

import "testing"

func FuzzClassify(f *testing.F) {
	f.Add("https://www.youtube.com/watch?v=dQw4w9WgXcQ")
	f.Add("https://youtu.be/dQw4w9WgXcQ?list=PL1234567890")
	f.Add("https://www.youtube.com/playlist?list=PL1234567890")
	f.Add("dQw4w9WgXcQ")
	f.Add("spotify:track:4cOdK2wGLETKBW3PvgPWqT")
	f.Add("")
	f.Fuzz(func(t *testing.T, input string) {
		got := Classify(input)
		if got.Kind == KindVideo && !isVideoID(got.VideoID) {
			t.Errorf("video kind with invalid id %q for input %q", got.VideoID, input)
		}
		if got.Kind == KindPlaylist && !isPlaylistID(got.PlaylistID) {
			t.Errorf("playlist kind with invalid id %q for input %q", got.PlaylistID, input)
		}
	})
}
