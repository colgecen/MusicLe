package youtube

import "testing"

func TestContainerFromMime(t *testing.T) {
	cases := map[string]string{
		`audio/webm; codecs="opus"`:     "webm",
		`audio/mp4; codecs="mp4a.40.2"`: "m4a",
		`audio/ogg; codecs="opus"`:      "ogg",
		`video/mp4`:                     "m4a",
		``:                              "webm",
	}
	for in, want := range cases {
		if got := ContainerFromMime(in); got != want {
			t.Errorf("ContainerFromMime(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestChooseAudioFormatPrefersOpus(t *testing.T) {
	v := &Video{
		Formats: []Format{
			{ItagNo: 140, MimeType: `audio/mp4; codecs="mp4a.40.2"`, Bitrate: 128000, AudioChannels: 2, AudioSampleRate: "44100"},
			{ItagNo: 251, MimeType: `audio/webm; codecs="opus"`, Bitrate: 160000, AudioChannels: 2, AudioSampleRate: "48000"},
			{ItagNo: 137, MimeType: `video/mp4`, Bitrate: 4000000, AudioChannels: 0},
		},
	}
	f, err := ChooseAudioFormat(v)
	if err != nil {
		t.Fatal(err)
	}
	if f.ItagNo != 251 {
		t.Errorf("chose itag %d, want 251", f.ItagNo)
	}
}

func TestChooseAudioFormatNone(t *testing.T) {
	v := &Video{Formats: []Format{{ItagNo: 137, MimeType: "video/mp4", Width: 1920}}}
	if _, err := ChooseAudioFormat(v); err == nil {
		t.Error("expected an error when no audio format exists")
	}
}

func TestTrackInfoFromVideo(t *testing.T) {
	v := &Video{
		ID:       "abc12345678",
		Title:    "Title",
		Author:   "Author",
		Duration: 190000000000, // 190s
		Thumbnails: Thumbnails{
			{URL: "https://img/1.jpg"},
		},
	}
	f := &Format{MimeType: `audio/webm; codecs="opus"`, ContentLength: 1234}
	ti := TrackInfoFromVideo(v, f, "PL", 3)
	if ti.Title != "Title" || ti.Artist != "Author" || ti.Playlist != "PL" || ti.TrackNum != 3 {
		t.Errorf("bad track info: %+v", ti)
	}
	if ti.Format != "webm" || ti.ContentLen != 1234 || ti.Thumbnail != "https://img/1.jpg" {
		t.Errorf("bad format/len/thumb: %+v", ti)
	}
	if ti.Source != "abc12345678" {
		t.Errorf("source = %q", ti.Source)
	}
}

func TestMP3Convertible(t *testing.T) {
	cases := []struct {
		name string
		f    Format
		want bool
	}{
		{"opus audio-only", Format{MimeType: `audio/webm; codecs="opus"`, AudioChannels: 2}, true},
		{"opus case-insensitive", Format{MimeType: `audio/WebM; codecs="Opus"`, AudioChannels: 2}, true},
		{"aac-lc audio-only", Format{MimeType: `audio/mp4; codecs="mp4a.40.2"`, AudioChannels: 2}, true},
		{"he-aac audio-only", Format{MimeType: `audio/mp4; codecs="mp4a.40.5"`, AudioChannels: 2}, false},
		{"combined A/V aac", Format{MimeType: `video/mp4; codecs="avc1.42001E, mp4a.40.2"`, AudioChannels: 2, Width: 640, Height: 360}, true},
		{"no audio", Format{MimeType: `video/webm; codecs="vp9"`}, false},
	}
	for _, c := range cases {
		if got := MP3Convertible(&c.f); got != c.want {
			t.Errorf("%s: MP3Convertible = %v, want %v", c.name, got, c.want)
		}
	}
	if MP3Convertible(nil) {
		t.Error("nil format should not be convertible")
	}
}
