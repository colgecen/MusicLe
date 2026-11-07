package youtube

import "testing"

func TestClassify(t *testing.T) {
	const (
		videoID = "dQw4w9WgXcQ"
		listID  = "PLFgquLnL59alCl_2TQvOiD5Vgm1hCaGSI"
	)
	cases := []struct {
		in       string
		kind     Kind
		video    string
		playlist string
	}{
		// Plain videos.
		{"https://www.youtube.com/watch?v=" + videoID, KindVideo, videoID, ""},
		{"https://youtu.be/" + videoID, KindVideo, videoID, ""},
		{"https://music.youtube.com/watch?v=" + videoID, KindVideo, videoID, ""},
		{"https://www.youtube.com/shorts/" + videoID, KindVideo, videoID, ""},
		{"https://www.youtube.com/embed/" + videoID, KindVideo, videoID, ""},
		{"https://www.youtube.com/live/" + videoID, KindVideo, videoID, ""},
		{videoID, KindVideo, videoID, ""},

		// A video that is being watched inside a playlist stays a video.
		{"https://www.youtube.com/watch?v=" + videoID + "&list=" + listID, KindVideo, videoID, listID},
		{"https://youtu.be/" + videoID + "?list=" + listID, KindVideo, videoID, listID},
		{"https://www.youtube.com/shorts/" + videoID + "?list=" + listID, KindVideo, videoID, listID},

		// Playlists.
		{"https://www.youtube.com/playlist?list=" + listID, KindPlaylist, "", listID},
		{"https://music.youtube.com/playlist?list=OLAK5uy_kABCDEFGHIJKLMNOP", KindPlaylist, "", "OLAK5uy_kABCDEFGHIJKLMNOP"},
		{listID, KindPlaylist, "", listID},
		{"UU" + "abcdefghijklmnopqrstuv", KindPlaylist, "", "UUabcdefghijklmnopqrstuv"},

		// Not YouTube references.
		{"https://open.spotify.com/track/abc", KindUnknown, "", ""},
		{"me at the zoo", KindUnknown, "", ""},
		{"", KindUnknown, "", ""},
		{"short", KindUnknown, "", ""},
	}
	for _, c := range cases {
		got := Classify(c.in)
		if got.Kind != c.kind {
			t.Errorf("Classify(%q).Kind = %v, want %v", c.in, got.Kind, c.kind)
			continue
		}
		if got.VideoID != c.video {
			t.Errorf("Classify(%q).VideoID = %q, want %q", c.in, got.VideoID, c.video)
		}
		if got.PlaylistID != c.playlist {
			t.Errorf("Classify(%q).PlaylistID = %q, want %q", c.in, got.PlaylistID, c.playlist)
		}
	}
}

func TestIsPlaylistURL(t *testing.T) {
	cases := map[string]bool{
		"https://www.youtube.com/playlist?list=PL1234567890abcdef": true,
		"https://www.youtube.com/watch?v=abc&list=PL123":           false,
		"https://youtu.be/dQw4w9WgXcQ?list=PL123":                  false,
		"PL1234567890abcdef":                                       true,
		"https://www.youtube.com/watch?v=jNQXAC9IVRw":              false,
	}
	for in, want := range cases {
		if got := IsPlaylistURL(in); got != want {
			t.Errorf("IsPlaylistURL(%q) = %v, want %v", in, got, want)
		}
	}
}
