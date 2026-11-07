package youtube

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearchYouTubeContextMockHTTP(t *testing.T) {
	body := `<html><script>var ytInitialData = {"contents":{"videoRenderer":{"videoId":"dQw4w9WgXcQ","title":{"runs":[{"text":"Rick Astley - Never Gonna Give You Up"}]},"ownerText":{"runs":[{"text":"Rick Astley"}]},"lengthText":{"simpleText":"3:33"}}}};</script></html>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/results") {
			t.Errorf("path = %q, want /results", r.URL.Path)
		}
		if got := r.URL.Query().Get("search_query"); got != "rick astley" {
			t.Errorf("search_query = %q", got)
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	old := youTubeBaseURL
	youTubeBaseURL = srv.URL
	defer func() { youTubeBaseURL = old }()

	results, err := SearchYouTubeContext(context.Background(), "rick astley", 5)
	if err != nil {
		t.Fatalf("SearchYouTubeContext: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if results[0].VideoID != "dQw4w9WgXcQ" || results[0].DurationSec != 213 {
		t.Errorf("result = %+v", results[0])
	}
}

func TestSearchYouTubeContextHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusTooManyRequests)
	}))
	defer srv.Close()

	old := youTubeBaseURL
	youTubeBaseURL = srv.URL
	defer func() { youTubeBaseURL = old }()

	_, err := SearchYouTubeContext(context.Background(), "x", 1)
	if err == nil || !strings.Contains(err.Error(), "HTTP 429") {
		t.Fatalf("error = %v, want HTTP 429", err)
	}
}
