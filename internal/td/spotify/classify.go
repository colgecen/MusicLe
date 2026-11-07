package spotify

import (
	"net/url"
	"strings"
)

// Kind is what a user-supplied Spotify reference points at.
type Kind int

const (
	// KindUnknown means the input is a search query or an id whose type is not
	// encoded in the input itself.
	KindUnknown Kind = iota
	KindTrack
	KindPlaylist
	KindAlbum
	KindArtist
)

func (k Kind) String() string {
	switch k {
	case KindTrack:
		return "track"
	case KindPlaylist:
		return "playlist"
	case KindAlbum:
		return "album"
	case KindArtist:
		return "artist"
	default:
		return "unknown"
	}
}

// Classify extracts the kind and id from a Spotify URL, URI or bare id.
//
// A bare 22-character id is reported as KindUnknown because Spotify uses the
// same id shape for tracks, playlists and albums; the caller must ask the API
// what it actually is.
func Classify(input string) (Kind, string) {
	s := strings.TrimSpace(input)
	if s == "" {
		return KindUnknown, ""
	}
	lower := strings.ToLower(s)

	// spotify:track:ID, spotify:album:ID, and the legacy
	// spotify:user:<user>:playlist:ID form.
	if strings.HasPrefix(lower, "spotify:") {
		parts := strings.Split(s, ":")
		for i := 0; i+1 < len(parts); i++ {
			if k, ok := kindFromWord(parts[i]); ok {
				if id := strings.TrimSpace(parts[i+1]); id != "" {
					return k, id
				}
				return KindUnknown, ""
			}
		}
		return KindUnknown, ""
	}

	// Users often paste "open.spotify.com/track/..." without the scheme.
	if strings.HasPrefix(lower, "open.spotify.com/") || strings.HasPrefix(lower, "play.spotify.com/") {
		s = "https://" + s
		lower = strings.ToLower(s)
	}

	if strings.Contains(s, "://") {
		u, err := url.Parse(s)
		if err != nil {
			return KindUnknown, ""
		}
		segs := splitPath(u.Path)
		for i := 0; i+1 < len(segs); i++ {
			if k, ok := kindFromWord(segs[i]); ok {
				if id := strings.TrimSpace(segs[i+1]); id != "" {
					return k, id
				}
				return KindUnknown, ""
			}
		}
		return KindUnknown, ""
	}

	if LooksLikeID(s) {
		return KindUnknown, s
	}
	return KindUnknown, ""
}

func kindFromWord(w string) (Kind, bool) {
	switch strings.ToLower(w) {
	case "track":
		return KindTrack, true
	case "playlist":
		return KindPlaylist, true
	case "album":
		return KindAlbum, true
	case "artist":
		return KindArtist, true
	default:
		return KindUnknown, false
	}
}

func splitPath(p string) []string {
	var out []string
	for _, seg := range strings.Split(p, "/") {
		if seg != "" {
			out = append(out, seg)
		}
	}
	return out
}

// LooksLikeID reports whether s is a bare 22-character Spotify id.
func LooksLikeID(s string) bool {
	if len(s) != 22 {
		return false
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

// IsCollection reports whether the input points at "Liked Songs", which is
// account-specific and therefore unavailable with client credentials.
func IsCollection(input string) bool {
	s := strings.ToLower(strings.TrimSpace(input))
	return strings.Contains(s, "/collection") || strings.Contains(s, ":collection")
}
