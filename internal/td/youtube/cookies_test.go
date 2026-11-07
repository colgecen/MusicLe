package youtube

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestNewCookieJarLoadsNetscapeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cookies.txt")
	content := "# Netscape HTTP Cookie File\n" +
		"#HttpOnly_.youtube.com\tTRUE\t/\tTRUE\t0\tSID\tabc123\n" +
		".youtube.com\tTRUE\t/\tFALSE\t0\tPREF\tf6=400\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	jar, err := newCookieJar(path)
	if err != nil {
		t.Fatalf("newCookieJar: %v", err)
	}
	u, _ := url.Parse("https://www.youtube.com/watch?v=x")
	got := map[string]string{}
	for _, c := range jar.Cookies(u) {
		got[c.Name] = c.Value
	}
	if got["SID"] != "abc123" {
		t.Errorf("SID = %q, want abc123", got["SID"])
	}
	if got["PREF"] != "f6=400" {
		t.Errorf("PREF = %q, want f6=400", got["PREF"])
	}
	if got["CONSENT"] == "" || got["SOCS"] == "" {
		t.Error("default consent cookies missing")
	}
}

func TestNewCookieJarMissingFileReturnsError(t *testing.T) {
	if _, err := newCookieJar(filepath.Join(t.TempDir(), "nope.txt")); err == nil {
		t.Error("expected an error for a missing cookie file")
	}
}

func TestSetCookieFileValidatesPath(t *testing.T) {
	old := cookieFilePath
	defer func() { cookieFilePath = old }()

	if err := SetCookieFile(filepath.Join(t.TempDir(), "missing.txt")); err == nil {
		t.Error("expected error for missing file")
	}
	if cookieFilePath != "" {
		t.Errorf("cookieFilePath = %q, want empty after failure", cookieFilePath)
	}

	path := filepath.Join(t.TempDir(), "cookies.txt")
	if err := os.WriteFile(path, []byte("# empty\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := SetCookieFile(path); err != nil {
		t.Fatalf("SetCookieFile: %v", err)
	}
	if cookieFilePath != path {
		t.Errorf("cookieFilePath = %q, want %q", cookieFilePath, path)
	}
}

func TestIsSorryURL(t *testing.T) {
	cases := []struct {
		raw  string
		want bool
	}{
		{"https://www.google.com/sorry/index?continue=x", true},
		{"https://www.youtube.com/results", false},
		{"https://example.com/sorry", false},
	}
	for _, c := range cases {
		u, _ := url.Parse(c.raw)
		if got := isSorryURL(u); got != c.want {
			t.Errorf("isSorryURL(%q) = %v, want %v", c.raw, got, c.want)
		}
	}
	if isSorryURL(nil) {
		t.Error("nil URL must not be a sorry URL")
	}
}

func TestHeaderTransportInjectsBrowserHeaders(t *testing.T) {
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tr := &headerTransport{base: http.DefaultTransport}
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()
	if got.Get("User-Agent") == "" {
		t.Error("User-Agent not injected")
	}
	if got.Get("Accept-Language") == "" {
		t.Error("Accept-Language not injected")
	}
	if got.Get("Sec-Fetch-Mode") != "navigate" {
		t.Errorf("Sec-Fetch-Mode = %q", got.Get("Sec-Fetch-Mode"))
	}
}
