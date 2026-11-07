package spotify

import "testing"

func TestParseID(t *testing.T) {
	cases := []struct {
		in   string
		kind string
		id   string
	}{
		{"https://open.spotify.com/track/4uLU6hMCjMI75M1A2tKUQC", "track", "4uLU6hMCjMI75M1A2tKUQC"},
		{"https://open.spotify.com/playlist/37i9dQZF1DXcBWIGoYBM5M", "playlist", "37i9dQZF1DXcBWIGoYBM5M"},
		{"https://open.spotify.com/intl-tr/track/abc123", "track", "abc123"},
		{"spotify:track:abc123", "track", "abc123"},
		{"spotify:album:xyz789", "album", "xyz789"},
		// A bare id carries no type; ResolveAny asks the API instead of
		// guessing "track".
		{"4uLU6hMCjMI75M1A2tKUQC", "unknown", "4uLU6hMCjMI75M1A2tKUQC"},
	}
	for _, c := range cases {
		kind, id := ParseID(c.in)
		if kind != c.kind || id != c.id {
			t.Errorf("ParseID(%q) = (%q,%q), want (%q,%q)", c.in, kind, id, c.kind, c.id)
		}
	}
}

func TestParseIDWithQuery(t *testing.T) {
	kind, id := ParseID("https://open.spotify.com/track/abc123?si=xyz")
	if kind != "track" || id != "abc123" {
		t.Errorf("got (%q,%q)", kind, id)
	}
}
