package youtube

import "testing"

func TestParseDurationText(t *testing.T) {
	cases := map[string]int{
		"3:47":    227,
		"1:02:33": 3753,
		"0:19":    19,
		"12:00":   720,
		"":        0,
		"abc":     0,
	}
	for in, want := range cases {
		if got := parseDurationText(in); got != want {
			t.Errorf("parseDurationText(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestNormalizeText(t *testing.T) {
	cases := map[string]string{
		"Şarkı Adı":                 "sarki adi",
		"AC/DC - Thunderstruck":     "ac dc thunderstruck",
		"Beyoncé — Halo (Official)": "beyonce halo official",
	}
	for in, want := range cases {
		if got := normalizeText(in); got != want {
			t.Errorf("normalizeText(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTitleTokensDropsNoise(t *testing.T) {
	got := titleTokens("Song Name (Official Video) HD")
	want := []string{"song", "name"}
	if len(got) != len(want) {
		t.Fatalf("titleTokens = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("titleTokens = %v, want %v", got, want)
		}
	}
}

func TestPickBestPrefersDurationMatch(t *testing.T) {
	candidates := []SearchResult{
		{VideoID: "wrong000001", Title: "Song Name (Live at Somewhere)", Channel: "Random Uploads", DurationSec: 480},
		{VideoID: "right000001", Title: "Artist - Song Name", Channel: "Artist - Topic", DurationSec: 212},
		{VideoID: "other000001", Title: "Song Name Remix", Channel: "DJ", DurationSec: 300},
	}
	best := PickBest(candidates, "Song Name", "Artist", 210)
	if best == nil {
		t.Fatal("nil result")
	}
	if best.VideoID != "right000001" {
		t.Errorf("picked %q, want right000001", best.VideoID)
	}
}

func TestPickBestPrefersTitleWhenNoDuration(t *testing.T) {
	candidates := []SearchResult{
		{VideoID: "aaaa0000001", Title: "Completely Different Thing", Channel: "x"},
		{VideoID: "bbbb0000001", Title: "Wanted Song Title", Channel: "y"},
	}
	best := PickBest(candidates, "Wanted Song Title", "", 0)
	if best == nil || best.VideoID != "bbbb0000001" {
		t.Errorf("picked %+v", best)
	}
}

func TestPickBestEmpty(t *testing.T) {
	if PickBest(nil, "x", "y", 0) != nil {
		t.Error("expected nil for empty candidates")
	}
}

func TestLooksLikeURL(t *testing.T) {
	cases := map[string]bool{
		"https://www.youtube.com/watch?v=jNQXAC9IVRw": true,
		"https://youtu.be/jNQXAC9IVRw":                true,
		"jNQXAC9IVRw":                                 true,
		"me at the zoo":                               false,
		"short":                                       false,
		"has space id":                                false,
	}
	for in, want := range cases {
		if got := LooksLikeURL(in); got != want {
			t.Errorf("LooksLikeURL(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestParseSearchResultsRealPayload(t *testing.T) {
	// Minimal slice of a real ytInitialData search response.
	body := []byte(`<script>var ytInitialData = {"contents":{"twoColumnSearchResultsRenderer":{"primaryContents":{"sectionListRenderer":{"contents":[{"itemSectionRenderer":{"contents":[{"videoRenderer":{"videoId":"jNQXAC9IVRw","title":{"runs":[{"text":"Me at the zoo"}]},"ownerText":{"runs":[{"text":"jawed"}]},"lengthText":{"simpleText":"0:19"}}},{"videoRenderer":{"videoId":"dQw4w9WgXcQ","title":{"runs":[{"text":"Rick Astley - Never Gonna Give You Up"}]},"ownerText":{"runs":[{"text":"Rick Astley"}]},"lengthText":{"simpleText":"3:33"}}}]}}]}}}}};</script>`)
	results := parseSearchResults(body)
	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}
	if results[0].VideoID != "jNQXAC9IVRw" || results[0].Title != "Me at the zoo" || results[0].Channel != "jawed" {
		t.Errorf("first = %+v", results[0])
	}
	if results[0].DurationSec != 19 {
		t.Errorf("duration = %d, want 19", results[0].DurationSec)
	}
	if results[1].DurationSec != 213 {
		t.Errorf("duration = %d, want 213", results[1].DurationSec)
	}
}

func TestSearchResultURL(t *testing.T) {
	r := SearchResult{VideoID: "abc"}
	if r.URL() != "https://www.youtube.com/watch?v=abc" {
		t.Errorf("URL = %q", r.URL())
	}
}
