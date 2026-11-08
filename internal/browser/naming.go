package browser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"MusicLe/state"
)

// NextProfileName derives an auto profile display name + folder for a platform.
// The first profile is "[Platform] Profili" (e.g. "Spotify Profili",
// "YouTube Profili"); subsequent ones are "[Platform] Profili 2", "… 3", etc.
// The folder name is slugified and made unique within profilesDir.
func NextProfileName(profiles []state.Profile, platform Platform, profilesDir string) (string, string) {
	prefix := "Spotify Profili"
	if platform == PlatformYouTube {
		prefix = "YouTube Profili"
	}
	n := 0
	for _, p := range profiles {
		if strings.HasPrefix(p.DisplayName, prefix) {
			n++
		}
	}
	displayName := prefix
	if n > 0 {
		displayName = fmt.Sprintf("%s %d", prefix, n+1)
	}
	folder := Slugify(displayName)
	base := folder
	i := 2
	for {
		if _, err := os.Stat(filepath.Join(profilesDir, folder)); os.IsNotExist(err) {
			break
		}
		folder = fmt.Sprintf("%s_%d", base, i)
		i++
	}
	return displayName, folder
}

// Slugify converts a display name into a filesystem-safe folder name,
// collapsing runs of separators into a single underscore.
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevSep := false
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevSep = false
		case r == ' ' || r == '-' || r == '_':
			if !prevSep {
				b.WriteRune('_')
				prevSep = true
			}
		default:
			prevSep = false
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		out = "playlist"
	}
	return out
}
