package spotify

import "testing"

func TestClassify(t *testing.T) {
	const (
		trackID    = "4cOdK2wGLETKBW3PvgPWqT"
		playlistID = "37i9dQZF1DXcBWIGoYBM5M"
		albumID    = "1ATL5GLyefJaxhQzSPVrLX"
	)
	cases := []struct {
		in   string
		kind Kind
		id   string
	}{
		{"https://open.spotify.com/track/" + trackID, KindTrack, trackID},
		{"https://open.spotify.com/intl-tr/track/" + trackID + "?si=abc", KindTrack, trackID},
		{"https://open.spotify.com/playlist/" + playlistID, KindPlaylist, playlistID},
		{"https://open.spotify.com/album/" + albumID, KindAlbum, albumID},
		{"https://open.spotify.com/artist/abc", KindArtist, "abc"},
		{"spotify:track:" + trackID, KindTrack, trackID},
		{"spotify:playlist:" + playlistID, KindPlaylist, playlistID},
		{"spotify:album:" + albumID, KindAlbum, albumID},
		// Legacy URI keeps its type.
		{"spotify:user:spotify:playlist:" + playlistID, KindPlaylist, playlistID},
		// A bare id is ambiguous, never assumed to be a track.
		{trackID, KindUnknown, trackID},
		// Scheme-less host, uppercase scheme and trailing slash/embed paths.
		{"open.spotify.com/track/" + trackID, KindTrack, trackID},
		{"https://open.spotify.com/embed/track/" + trackID + "/", KindTrack, trackID},
		{"SPOTIFY:TRACK:" + trackID, KindTrack, trackID},
		{"https://play.spotify.com/album/" + albumID, KindAlbum, albumID},
		{"https://open.spotify.com/user/someone/playlist/" + playlistID, KindPlaylist, playlistID},
		// Empty ids must not classify as a real reference.
		{"spotify:track:", KindUnknown, ""},
		{"https://open.spotify.com/track/", KindUnknown, ""},
		// Free text and unsupported links are not resolved here.
		{"never gonna give you up", KindUnknown, ""},
		{"", KindUnknown, ""},
	}
	for _, c := range cases {
		kind, id := Classify(c.in)
		if kind != c.kind || id != c.id {
			t.Errorf("Classify(%q) = (%v, %q), want (%v, %q)", c.in, kind, id, c.kind, c.id)
		}
	}
}

func TestLooksLikeID(t *testing.T) {
	if !LooksLikeID("4cOdK2wGLETKBW3PvgPWqT") {
		t.Error("valid id rejected")
	}
	for _, s := range []string{"", "short", "has space here 12345", "4cOdK2wGLETKBW3PvgPWqT!", "4cOdK2wGLETKBW3PvgPWq"} {
		if LooksLikeID(s) {
			t.Errorf("LooksLikeID(%q) = true, want false", s)
		}
	}
}

func TestIsCollection(t *testing.T) {
	if !IsCollection("https://open.spotify.com/collection/tracks") {
		t.Error("collection link not detected")
	}
	if IsCollection("https://open.spotify.com/playlist/abc") {
		t.Error("playlist misdetected as collection")
	}
}

func TestParseIDCompatibility(t *testing.T) {
	kind, id := ParseID("https://open.spotify.com/playlist/" + "37i9dQZF1DXcBWIGoYBM5M")
	if kind != "playlist" || id != "37i9dQZF1DXcBWIGoYBM5M" {
		t.Errorf("ParseID = (%q, %q)", kind, id)
	}
}
