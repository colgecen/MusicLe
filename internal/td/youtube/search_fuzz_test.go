package youtube

import "testing"

func FuzzParseSearchResults(f *testing.F) {
	f.Add([]byte(`<script>var ytInitialData = {"contents":{"videoRenderer":{"videoId":"dQw4w9WgXcQ","title":{"runs":[{"text":"Song"}]}}}};</script>`))
	f.Add([]byte(`"videoId":"jNQXAC9IVRw"`))
	f.Add([]byte(`<html></html>`))
	f.Add([]byte(`var ytInitialData = {};</script>`))
	f.Fuzz(func(t *testing.T, body []byte) {
		// The parser must never panic on arbitrary server responses.
		_ = parseSearchResults(body)
	})
}
