package download

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"MusicLeCLI/internal/td/model"
)

func TestExtFromContainer(t *testing.T) {
	cases := map[string]string{
		"m4a":  "m4a",
		"mp4":  "m4a",
		"opus": "opus",
		"ogg":  "ogg",
		"webm": "webm",
		"":     "webm",
		"xyz":  "webm",
	}
	for in, want := range cases {
		if got := extFromContainer(in); got != want {
			t.Errorf("extFromContainer(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildFileName(t *testing.T) {
	cases := []struct{ artist, title, id, container, want string }{
		{"Artist", "Song", "dQw4w9WgXcQ", "webm", "Artist - Song [dQw4w9WgXcQ].webm"},
		{"", "Song", "dQw4w9WgXcQ", "m4a", "Song [dQw4w9WgXcQ].m4a"},
		{"Unknown", "Song", "abc", "webm", "Song [abc].webm"},
		{"A/B", "C:D", "id", "webm", "A_B - C_D [id].webm"},
		{"", "", "id", "webm", "id [id].webm"},
	}
	for _, c := range cases {
		got := BuildFileName(c.artist, c.title, c.id, c.container)
		if got != c.want {
			t.Errorf("BuildFileName(%q,%q,%q,%q) = %q, want %q", c.artist, c.title, c.id, c.container, got, c.want)
		}
	}
}

func TestBuildFileNameLongTitle(t *testing.T) {
	long := strings.Repeat("a", 200)
	got := BuildFileName("", long, "id", "webm")
	name := strings.TrimSuffix(got, ".webm")
	if len([]rune(name)) > 140 {
		t.Errorf("name too long: %d", len([]rune(name)))
	}
}

func TestStartsWithArtist(t *testing.T) {
	cases := []struct {
		title, artist string
		want          bool
	}{
		{"Rick Astley - Never Gonna Give You Up", "Rick Astley", true},
		{"Şarkı Adı", "Şarkı", true},
		{"Never Gonna Give You Up", "Rick Astley", false},
		{"Anyone", "", false},
	}
	for _, c := range cases {
		if got := startsWithArtist(c.title, c.artist); got != c.want {
			t.Errorf("startsWithArtist(%q, %q) = %v, want %v", c.title, c.artist, got, c.want)
		}
	}
}

func TestArtistTitle(t *testing.T) {
	if got := ArtistTitle("Rick Astley", "Rick Astley - Never Gonna Give You Up"); got != "Rick Astley - Never Gonna Give You Up" {
		t.Errorf("got %q", got)
	}
	if got := ArtistTitle("Rick Astley", "Never Gonna Give You Up"); got != "Rick Astley - Never Gonna Give You Up" {
		t.Errorf("got %q", got)
	}
	if got := ArtistTitle("Unknown", "Song"); got != "Song" {
		t.Errorf("got %q", got)
	}
}

func TestTagTitleStripsArtistPrefix(t *testing.T) {
	cases := []struct{ artist, title, want string }{
		{"Rick Astley", "Rick Astley - Never Gonna Give You Up", "Never Gonna Give You Up"},
		{"jawed", "Me at the zoo", "Me at the zoo"},
		{"Artist", "Artist: Song", "Song"},
		{"", "Title", "Title"},
	}
	for _, c := range cases {
		if got := tagTitle(c.artist, c.title); got != c.want {
			t.Errorf("tagTitle(%q, %q) = %q, want %q", c.artist, c.title, got, c.want)
		}
	}
}

func TestMaybeConvertToMP3Disabled(t *testing.T) {
	SetMP3Output(false)
	defer SetMP3Output(false)
	got, err := maybeConvertToMP3(context.Background(), &model.TrackInfo{}, "x.webm", nil)
	if err != nil || got != "x.webm" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestConvertibleReason(t *testing.T) {
	dir := t.TempDir()
	mp3 := filepath.Join(dir, "a.mp3")
	if err := os.WriteFile(mp3, []byte("ID3"), 0644); err != nil {
		t.Fatal(err)
	}
	if ok, _ := ConvertibleReason(mp3); ok {
		t.Error("mp3 should not be converted again")
	}
	webm := filepath.Join(dir, "a.webm")
	if err := os.WriteFile(webm, []byte{0x1a, 0x45, 0xdf, 0xa3, 0}, 0644); err != nil {
		t.Fatal(err)
	}
	if ok, err := ConvertibleReason(webm); !ok || err != nil {
		t.Errorf("webm should be convertible, got %v %v", ok, err)
	}
	// An M4A is a real MP4 container (ftyp at bytes 4-8) and is convertible.
	m4a := filepath.Join(dir, "a.m4a")
	if err := os.WriteFile(m4a, []byte{0, 0, 0, 0x1c, 'f', 't', 'y', 'p', 'M', '4', 'A', ' '}, 0644); err != nil {
		t.Fatal(err)
	}
	if ok, err := ConvertibleReason(m4a); !ok || err != nil {
		t.Errorf("m4a should be convertible, got %v %v", ok, err)
	}
	// Something that is neither WebM nor MP4 stays rejected.
	other := filepath.Join(dir, "a.flac")
	if err := os.WriteFile(other, []byte("fLaC"), 0644); err != nil {
		t.Fatal(err)
	}
	if ok, _ := ConvertibleReason(other); ok {
		t.Error("flac should be rejected")
	}
}

func TestScaleProgress(t *testing.T) {
	var got []float64
	p := scaleProgress(func(v float64) { got = append(got, v) }, 70, 100)
	p(0)
	p(50)
	p(100)
	p(200) // clamped
	want := []float64{70, 85, 100, 100}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %v, want %v", i, got[i], want[i])
		}
	}
	if scaleProgress(nil, 0, 70) != nil {
		t.Error("nil callback should stay nil")
	}
}

func TestIsGatedError(t *testing.T) {
	for _, msg := range []string{"unexpected status code: 403", "cipher not found", "error parsing signature tokens"} {
		if !isGatedError(errors.New(msg)) {
			t.Errorf("%q should be gated", msg)
		}
	}
	if isGatedError(errors.New("connection reset by peer")) {
		t.Error("network error should not be gated")
	}
	if isGatedError(nil) {
		t.Error("nil should not be gated")
	}
}

func TestDescribeDownloadFailure(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{errors.New("cipher not found"), "PO token"},
		{errors.New("unexpected status code: 403"), "PO token"},
		{errors.New("baska bir hata"), "baska bir hata"},
	}
	for _, c := range cases {
		got := describeDownloadFailure(4, c.err).Error()
		if !strings.Contains(got, c.want) {
			t.Errorf("describeDownloadFailure(%v) = %q, want it to contain %q", c.err, got, c.want)
		}
	}
}
