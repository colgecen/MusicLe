package youtube

import (
	"net/url"
	"strings"
)

// Kind is what a user-supplied YouTube reference points at.
type Kind int

const (
	// KindUnknown means the input is neither a video nor a playlist reference.
	KindUnknown Kind = iota
	// KindVideo means the input refers to exactly one video.
	KindVideo
	// KindPlaylist means the input refers to a playlist.
	KindPlaylist
)

func (k Kind) String() string {
	switch k {
	case KindVideo:
		return "video"
	case KindPlaylist:
		return "playlist"
	default:
		return "unknown"
	}
}

// Target is the result of classifying a YouTube reference.
type Target struct {
	Kind Kind
	// VideoID is set when a video id could be extracted.
	VideoID string
	// PlaylistID is set when a list id was present (even for a watch URL).
	PlaylistID string
}

// playlistIDPrefixes are the prefixes YouTube uses for playlist/album/mix ids.
// They are long enough that they can never collide with an 11-char video id.
var playlistIDPrefixes = []string{
	"PL",       // regular playlist
	"UU",       // channel uploads
	"LL",       // liked videos
	"FL",       // favourites
	"WL",       // watch later
	"RD",       // mix / radio
	"VL",       // auto-generated
	"OLAK5uy_", // album playlist
}

// Classify decides whether input points at a single video or at a playlist.
//
// A watch/shorts/youtu.be link is always a video, even when it carries a
// `list=` parameter: that parameter only says which playlist the video is
// being viewed from, not that the whole list was requested.
func Classify(input string) Target {
	s := strings.TrimSpace(input)
	if s == "" {
		return Target{Kind: KindUnknown}
	}

	if strings.Contains(s, "://") {
		return classifyURL(s)
	}

	// spotify-style URIs and other schemes are not YouTube references.
	if strings.Contains(s, ":") {
		return Target{Kind: KindUnknown}
	}

	if isVideoID(s) {
		return Target{Kind: KindVideo, VideoID: s}
	}
	if isPlaylistID(s) {
		return Target{Kind: KindPlaylist, PlaylistID: s}
	}
	return Target{Kind: KindUnknown}
}

func classifyURL(raw string) Target {
	u, err := url.Parse(raw)
	if err != nil {
		return Target{Kind: KindUnknown}
	}
	host := strings.ToLower(u.Hostname())
	host = strings.TrimPrefix(host, "www.")
	host = strings.TrimPrefix(host, "music.")
	host = strings.TrimPrefix(host, "m.")
	if host != "youtube.com" && host != "youtu.be" && host != "youtube-nocookie.com" {
		return Target{Kind: KindUnknown}
	}

	listID := u.Query().Get("list")
	segs := splitPath(u.Path)

	// youtu.be/<id> and youtube.com/shorts|embed|live|v/<id>.
	if host == "youtu.be" && len(segs) > 0 && isVideoID(segs[0]) {
		return Target{Kind: KindVideo, VideoID: segs[0], PlaylistID: listID}
	}
	for _, key := range []string{"shorts", "embed", "live", "v"} {
		for i, seg := range segs {
			if seg == key && i+1 < len(segs) && isVideoID(segs[i+1]) {
				return Target{Kind: KindVideo, VideoID: segs[i+1], PlaylistID: listID}
			}
		}
	}

	if v := u.Query().Get("v"); isVideoID(v) {
		return Target{Kind: KindVideo, VideoID: v, PlaylistID: listID}
	}
	// /playlist?list=... (also /watch with only a list id).
	if isPlaylistID(listID) {
		return Target{Kind: KindPlaylist, PlaylistID: listID}
	}
	return Target{Kind: KindUnknown}
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

// isVideoID reports whether s is an 11-character YouTube video id.
func isVideoID(s string) bool {
	if len(s) != 11 {
		return false
	}
	for _, r := range s {
		if !(r == '-' || r == '_' ||
			(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

// isPlaylistID reports whether s looks like a YouTube playlist id.
func isPlaylistID(s string) bool {
	if len(s) < 12 {
		return false
	}
	for _, p := range playlistIDPrefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
