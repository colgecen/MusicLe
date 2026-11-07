package youtube

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestDownloadStreamResumesWithRange(t *testing.T) {
	payload := bytes.Repeat([]byte("0123456789"), 100) // 1000 bytes
	const resumeAt = 400
	var sawRange string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawRange = r.Header.Get("Range")
		start := 0
		if rg := r.Header.Get("Range"); strings.HasPrefix(rg, "bytes=") {
			start, _ = strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(rg, "bytes="), "-"))
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, len(payload)-1, len(payload)))
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)-start))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(payload[start:])
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "song.webm.part")
	if err := os.WriteFile(dest, payload[:resumeAt], 0644); err != nil {
		t.Fatal(err)
	}
	var last float64
	err := downloadStream(context.Background(), srv.Client(), srv.URL, int64(len(payload)), dest, func(p float64) { last = p })
	if err != nil {
		t.Fatalf("downloadStream: %v", err)
	}
	if sawRange != "bytes=400-" {
		t.Errorf("Range header = %q, want bytes=400-", sawRange)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("file mismatch: got %d bytes", len(got))
	}
	if last != 100 {
		t.Errorf("last progress = %v, want 100", last)
	}
}

func TestDownloadStreamRestartsWhenServerIgnoresRange(t *testing.T) {
	payload := bytes.Repeat([]byte("abcdefghij"), 50)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "song.webm.part")
	// Stale bytes from an older attempt must be replaced, not appended.
	if err := os.WriteFile(dest, []byte("stale"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := downloadStream(context.Background(), srv.Client(), srv.URL, int64(len(payload)), dest, nil); err != nil {
		t.Fatalf("downloadStream: %v", err)
	}
	got, _ := os.ReadFile(dest)
	if !bytes.Equal(got, payload) {
		t.Fatalf("file mismatch: got %d bytes, want %d", len(got), len(payload))
	}
}

func TestDownloadStreamCompleteSkipsRequest(t *testing.T) {
	payload := bytes.Repeat([]byte("x"), 100)
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "done.webm")
	if err := os.WriteFile(dest, payload, 0644); err != nil {
		t.Fatal(err)
	}
	last := -1.0
	if err := downloadStream(context.Background(), srv.Client(), srv.URL, int64(len(payload)), dest, func(p float64) { last = p }); err != nil {
		t.Fatalf("downloadStream: %v", err)
	}
	if called {
		t.Error("server was called for an already complete file")
	}
	if last != 100 {
		t.Errorf("progress = %v, want 100", last)
	}
}

func TestDownloadStreamSizeMismatch(t *testing.T) {
	payload := []byte("short")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "song.webm.part")
	err := downloadStream(context.Background(), srv.Client(), srv.URL, int64(len(payload)+10), dest, nil)
	if err == nil || !strings.Contains(err.Error(), "eksik indirme") {
		t.Fatalf("error = %v, want eksik indirme", err)
	}
}

func TestParseContentRange(t *testing.T) {
	cases := []struct {
		in    string
		start int64
		total int64
		ok    bool
	}{
		{"bytes 400-999/1000", 400, 1000, true},
		{"bytes 0-0/1", 0, 1, true},
		{"bytes 5-9/*", 5, 0, true},
		{"", 0, 0, false},
		{"items 1-2/3", 0, 0, false},
		{"bytes x-9/10", 0, 0, false},
	}
	for _, c := range cases {
		start, total, ok := parseContentRange(c.in)
		if start != c.start || total != c.total || ok != c.ok {
			t.Errorf("parseContentRange(%q) = (%d,%d,%v), want (%d,%d,%v)", c.in, start, total, ok, c.start, c.total, c.ok)
		}
	}
}
