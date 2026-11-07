package spotify

import "testing"

func FuzzClassify(f *testing.F) {
	f.Add("https://open.spotify.com/track/4cOdK2wGLETKBW3PvgPWqT")
	f.Add("https://open.spotify.com/intl-tr/playlist/37i9dQZF1DXcBWIGoYBM5M?si=x")
	f.Add("spotify:user:spotify:playlist:37i9dQZF1DXcBWIGoYBM5M")
	f.Add("open.spotify.com/album/1ATL5GLyefJaxhQzSPVrLX")
	f.Add("4cOdK2wGLETKBW3PvgPWqT")
	f.Add("")
	f.Fuzz(func(t *testing.T, input string) {
		kind, id := Classify(input)
		if id != "" && kind == KindUnknown && !LooksLikeID(id) {
			// Unknown kind may carry only a bare id; anything else is a bug.
			if len(id) != 22 {
				t.Errorf("unexpected id %q for input %q", id, input)
			}
		}
	})
}
