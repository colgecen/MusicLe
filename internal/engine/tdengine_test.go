package engine

import (
	"strings"
	"testing"
)

// TestDownloadWithTDRejectsEmptyInput verifies the guard that used to live in
// the yt-dlp wrapper: no URLs means a controlled error, not a silent no-op.
func TestDownloadWithTDRejectsEmptyInput(t *testing.T) {
	err := downloadWithTD(nil, t.TempDir(), true, nil)
	if err == nil || !strings.Contains(err.Error(), "URL") {
		t.Fatalf("beklenen hatal alinmadi: %v", err)
	}
}

// TestApplyCookieFilePropagatesBadPath verifies MUSICLE_COOKIES is honoured
// and a broken path fails loudly instead of downloading anonymously.
func TestApplyCookieFilePropagatesBadPath(t *testing.T) {
	t.Setenv(cookieEnvVar, "")
	if err := applyCookieFile(); err != nil {
		t.Fatalf("bos cookies yolu sessiz gecilmeliydi: %v", err)
	}
	t.Setenv(cookieEnvVar, "/nonexistent/dir/cookies.txt")
	if err := applyCookieFile(); err == nil {
		t.Fatal("olmayan cookies dosyasi hata vermeli")
	}
	err := downloadWithTD([]string{"https://www.youtube.com/watch?v=dQw4w9WgXcQ"}, t.TempDir(), true, nil)
	if err == nil || !strings.Contains(err.Error(), "cookies") {
		t.Fatalf("indirme cookies hatasiyla durmeliydi: %v", err)
	}
}

// TestStrictModeRejectsNonYouTubeInput guards the old yt-dlp-era behaviour:
// free text on the YouTube entry point must fail instead of silently
// downloading whatever a search happens to return.
func TestStrictModeRejectsNonYouTubeInput(t *testing.T) {
	t.Setenv(cookieEnvVar, "")
	for _, bad := range []string{"not-a-url", "acoustic likes bfdsa908", "https://example.com/watch?v=abc"} {
		err := downloadWithTD([]string{bad}, t.TempDir(), true, nil)
		if err == nil || !strings.Contains(err.Error(), "gecerli bir YouTube") {
			t.Fatalf("%q icin beklenen hata alinmadi: %v", bad, err)
		}
	}
}

// TestNormalizeYouTubeURLs keeps URL shapes untouched but drops blanks.
func TestNormalizeYouTubeURLs(t *testing.T) {
	in := []string{"  https://youtu.be/abc ", "", "\t", "dQw4w9WgXcQ"}
	got := normalizeYouTubeURLs(in)
	if len(got) != 2 || got[0] != "https://youtu.be/abc" || got[1] != "dQw4w9WgXcQ" {
		t.Fatalf("normalizeYouTubeURLs beklenmedik sonuc: %#v", got)
	}
}
