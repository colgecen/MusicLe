package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// SearchResult is one video returned by a YouTube search.
type SearchResult struct {
	VideoID     string
	Title       string
	Channel     string
	DurationSec int
}

// URL returns the canonical watch URL for the result.
func (r SearchResult) URL() string {
	return "https://www.youtube.com/watch?v=" + r.VideoID
}

var (
	ytInitialDataPattern = regexp.MustCompile(`var ytInitialData\s*=\s*(\{.+?\});</script>`)
	durationPattern      = regexp.MustCompile(`^(?:(\d+):)?(\d{1,2}):(\d{2})$`)
)

// youTubeBaseURL is a variable so tests can point the search client at an
// httptest server.
var youTubeBaseURL = "https://www.youtube.com"

// SearchYouTubeContext returns up to limit search results for a query.
func SearchYouTubeContext(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 8
	}
	q := url.Values{}
	q.Set("search_query", query)
	q.Set("hl", "en")
	u := youTubeBaseURL + "/results?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	client := clientWithCookies()
	resp, err := client.Do(req)
	if err != nil {
		return nil, localizeError(err)
	}
	defer resp.Body.Close()
	if resp.Request != nil && isSorryURL(resp.Request.URL) {
		return nil, fmt.Errorf("YouTube bot dogrulamasi istedi; --cookies ile kendi oturumunuzu kullanin")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("arama istegi basarisiz (HTTP %d)", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	results := parseSearchResults(body)
	if len(results) == 0 {
		return nil, fmt.Errorf("%q icin sonuc bulunamadi", query)
	}
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// parseSearchResults extracts videoRenderer entries from the search page.
func parseSearchResults(body []byte) []SearchResult {
	m := ytInitialDataPattern.FindSubmatch(body)
	if len(m) < 2 {
		// Fallback: plain videoId scrape (older/mobile layouts).
		re := regexp.MustCompile(`"videoId":"([A-Za-z0-9_-]{11})"`)
		seen := map[string]bool{}
		var out []SearchResult
		for _, match := range re.FindAllStringSubmatch(string(body), -1) {
			id := match[1]
			if seen[id] {
				continue
			}
			seen[id] = true
			out = append(out, SearchResult{VideoID: id})
		}
		return out
	}
	var data map[string]interface{}
	if err := json.Unmarshal(m[1], &data); err != nil {
		return nil
	}

	var out []SearchResult
	seen := map[string]bool{}
	walkSearchJSON(data, func(renderer map[string]interface{}) {
		id, _ := renderer["videoId"].(string)
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		r := SearchResult{VideoID: id}
		r.Title = runsText(renderer["title"])
		r.Channel = runsText(renderer["ownerText"])
		if r.Channel == "" {
			r.Channel = runsText(renderer["longBylineText"])
		}
		if lt, ok := renderer["lengthText"].(map[string]interface{}); ok {
			if s, ok := lt["simpleText"].(string); ok {
				r.DurationSec = parseDurationText(s)
			}
		}
		out = append(out, r)
	})
	return out
}

// walkSearchJSON recursively visits every videoRenderer object.
func walkSearchJSON(v interface{}, visit func(map[string]interface{})) {
	switch node := v.(type) {
	case map[string]interface{}:
		if r, ok := node["videoRenderer"].(map[string]interface{}); ok {
			visit(r)
		}
		for _, child := range node {
			walkSearchJSON(child, visit)
		}
	case []interface{}:
		for _, child := range node {
			walkSearchJSON(child, visit)
		}
	}
}

// runsText joins the "runs[].text" array of a YouTube text node.
func runsText(v interface{}) string {
	node, ok := v.(map[string]interface{})
	if !ok {
		return ""
	}
	runs, ok := node["runs"].([]interface{})
	if !ok {
		if s, ok := node["simpleText"].(string); ok {
			return s
		}
		return ""
	}
	var b strings.Builder
	for _, r := range runs {
		run, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		if s, ok := run["text"].(string); ok {
			b.WriteString(s)
		}
	}
	return strings.TrimSpace(b.String())
}

// parseDurationText converts "4:12" or "1:02:33" to seconds.
func parseDurationText(s string) int {
	s = strings.TrimSpace(s)
	m := durationPattern.FindStringSubmatch(s)
	if len(m) != 4 {
		return 0
	}
	hours, _ := strconv.Atoi(m[1])
	mins, _ := strconv.Atoi(m[2])
	secs, _ := strconv.Atoi(m[3])
	return hours*3600 + mins*60 + secs
}

// specialFolds maps letters that NFD does not decompose into ASCII.
var specialFolds = strings.NewReplacer(
	"ı", "i", "İ", "i", "ş", "s", "Ş", "s", "ğ", "g", "Ğ", "g",
	"ç", "c", "Ç", "c", "ö", "o", "Ö", "o", "ü", "u", "Ü", "u",
	"ñ", "n", "Ñ", "n", "ø", "o", "Ø", "o", "æ", "ae", "Æ", "ae",
	"ß", "ss", "ł", "l", "Ł", "l", "đ", "d", "Đ", "d", "ð", "d",
)

// normalizeText lowercases, strips accents and punctuation, and collapses spaces.
func normalizeText(s string) string {
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = specialFolds.Replace(s)
	s = norm.NFD.String(s)
	var b strings.Builder
	for _, r := range s {
		if unicode.Is(unicode.Mn, r) {
			continue // drop diacritics
		}
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
		default:
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// noiseWords are common YouTube upload suffixes that hurt title matching.
var noiseWords = map[string]bool{
	"official": true, "video": true, "audio": true, "lyrics": true, "lyric": true,
	"hd": true, "hq": true, "mv": true, "m": true, "vevo": true, "topic": true,
	"remaster": true, "remastered": true, "explicit": true, "4k": true,
	"visualizer": true, "clip": true, "music": true, "the": true,
}

// titleTokens returns the meaningful tokens of a string.
func titleTokens(s string) []string {
	var out []string
	for _, t := range strings.Fields(normalizeText(s)) {
		if noiseWords[t] || len(t) < 2 {
			continue
		}
		out = append(out, t)
	}
	return out
}

// PickBest chooses the search result that best matches the wanted title,
// artist and duration. Duration is the strongest signal: a track whose length
// is within a few seconds of the wanted one is almost certainly the same song.
// Returns nil only when there is nothing to choose from.
func PickBest(candidates []SearchResult, wantTitle, wantArtist string, wantDurationSec int) *SearchResult {
	if len(candidates) == 0 {
		return nil
	}
	type scored struct {
		r     SearchResult
		score float64
	}
	wantTokens := titleTokens(wantTitle)
	wantArtistTokens := titleTokens(wantArtist)

	scoredList := make([]scored, 0, len(candidates))
	for _, c := range candidates {
		if c.VideoID == "" {
			continue
		}
		score := 0.0
		title := normalizeText(c.Title)
		channel := normalizeText(c.Channel)
		hay := title + " " + channel

		// Title token overlap.
		if len(wantTokens) > 0 {
			matched := 0
			for _, t := range wantTokens {
				if strings.Contains(title, t) {
					matched++
				}
			}
			score += 6.0 * float64(matched) / float64(len(wantTokens))
		}
		// Artist presence anywhere in title/channel.
		if len(wantArtistTokens) > 0 {
			matched := 0
			for _, t := range wantArtistTokens {
				if strings.Contains(hay, t) {
					matched++
				}
			}
			score += 4.0 * float64(matched) / float64(len(wantArtistTokens))
		}
		// Duration agreement (decisive when known).
		if wantDurationSec > 0 && c.DurationSec > 0 {
			diff := c.DurationSec - wantDurationSec
			if diff < 0 {
				diff = -diff
			}
			switch {
			case diff <= 2:
				score += 10
			case diff <= 5:
				score += 7
			case diff <= 10:
				score += 4
			case diff <= 20:
				score += 1
			default:
				score -= 8 // clearly a different recording
			}
		}
		// Prefer official artist channels (Topic/VEVO) for audio quality.
		if strings.Contains(channel, "topic") || strings.Contains(channel, "vevo") {
			score += 1.5
		}
		scoredList = append(scoredList, scored{r: c, score: score})
	}
	if len(scoredList) == 0 {
		return nil
	}
	sort.SliceStable(scoredList, func(i, j int) bool { return scoredList[i].score > scoredList[j].score })
	best := scoredList[0].r
	return &best
}

// LooksLikeURL reports whether input is an http(s) URL or a bare video id.
func LooksLikeURL(input string) bool {
	s := strings.TrimSpace(input)
	if strings.Contains(s, "://") {
		return true
	}
	if len(s) != 11 {
		return false
	}
	for _, r := range s {
		if !(r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}
