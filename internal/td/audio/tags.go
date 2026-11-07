package audio

import (
	"strings"

	"github.com/bogem/id3v2/v2"
)

// TagUpdate holds the ID3v2 text values to write. Empty fields keep the value
// that is already stored in the file, so callers can update only one field.
type TagUpdate struct {
	Title  string
	Artist string
	Album  string
}

// WriteMP3Tags rewrites ID3v2 text frames in an MP3 file. A non-nil cover
// replaces the current front cover; a nil cover leaves the existing picture
// untouched.
func WriteMP3Tags(path string, up TagUpdate, cover []byte, coverMIME string) error {
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		return err
	}
	defer tag.Close()

	tag.SetVersion(4)
	if up.Title != "" {
		tag.SetTitle(up.Title)
	}
	if up.Artist != "" {
		tag.SetArtist(up.Artist)
	}
	if up.Album != "" {
		tag.SetAlbum(up.Album)
	}
	if cover != nil {
		tag.DeleteFrames("APIC")
		mime := coverMIME
		if mime == "" {
			mime = "image/jpeg"
		}
		tag.AddAttachedPicture(id3v2.PictureFrame{
			Encoding:    id3v2.EncodingUTF8,
			MimeType:    mime,
			PictureType: id3v2.PTFrontCover,
			Description: "Cover",
			Picture:     cover,
		})
	}
	return tag.Save()
}

// SplitArtistTitle derives an artist and title from a file name in the common
// "Artist - Title [videoID]" shape. The video id suffix is removed.
func SplitArtistTitle(name string) (artist, title string) {
	base := strings.TrimSuffix(name, filepathExt(name))
	base = youtubeIDSuffix.ReplaceAllString(base, "")
	base = strings.TrimSpace(base)
	if i := strings.Index(base, " - "); i > 0 {
		artist = strings.TrimSpace(base[:i])
		title = strings.TrimSpace(base[i+3:])
	}
	if title == "" {
		title = base
	}
	return artist, title
}

// filepathExt is a tiny local helper so SplitArtistTitle does not need the
// filepath package in every caller.
func filepathExt(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return name[i:]
		}
		if name[i] == '/' || name[i] == '\\' {
			break
		}
	}
	return ""
}
