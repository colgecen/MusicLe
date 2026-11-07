package util

import (
	"strings"
	"testing"
)

func TestSanitizeName(t *testing.T) {
	cases := map[string]string{
		"a/b:c*d?e\"f<g>h|i": "a_b_c_d_e_f_g_h_i",
		"  normal name  ":    "normal name",
		"Türkçe Şarkı":       "Türkçe Şarkı",
		// Windows rules: no trailing dot/space, no reserved device names.
		"name. ":    "name",
		"name...":   "name",
		"CON":       "_CON",
		"CON.mp3":   "_CON.mp3",
		"con":       "_con",
		"NUL.txt":   "_NUL.txt",
		"console":   "console",
		"normal.md": "normal.md",
	}
	for in, want := range cases {
		if got := SanitizeName(in); got != want {
			t.Errorf("SanitizeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFmtDuration(t *testing.T) {
	cases := map[float64]string{0: "00:00", 9: "00:09", 61: "01:01", 3599: "59:59", 3600: "60:00"}
	for in, want := range cases {
		if got := FmtDuration(in); got != want {
			t.Errorf("FmtDuration(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeDuration(t *testing.T) {
	cases := map[string]string{
		"":     "00:00",
		"1:02": "1:02",
		"65":   "01:05",
		"0":    "00:00",
		"abc":  "abc",
	}
	for in, want := range cases {
		if got := NormalizeDuration(in); got != want {
			t.Errorf("NormalizeDuration(%q) = %q, want %q", in, got, want)
		}
	}
}

func FuzzSanitizeName(f *testing.F) {
	f.Add("a/b:c*d?e\"f<g>h|i")
	f.Add("normal name")
	f.Fuzz(func(t *testing.T, in string) {
		out := SanitizeName(in)
		// Sanitized output must not contain unsafe characters.
		for _, r := range out {
			switch r {
			case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
				t.Fatalf("unsafe char %q in %q", r, out)
			}
		}
		if strings.HasSuffix(out, ".") || strings.HasSuffix(out, " ") {
			t.Fatalf("trailing dot/space in %q", out)
		}
	})
}

func TestFormatFromExt(t *testing.T) {
	cases := map[string]string{
		".mp3":  "MP3",
		".MP3":  "MP3",
		".flac": "FLAC",
		".m4a":  "AAC",
		".aac":  "AAC",
		".ogg":  "OGG",
		".wav":  "WAV",
		".opus": "Opus",
		".webm": "Opus",
		".txt":  "",
	}
	for in, want := range cases {
		if got := FormatFromExt(in); got != want {
			t.Errorf("FormatFromExt(%q) = %q, want %q", in, got, want)
		}
	}
}
