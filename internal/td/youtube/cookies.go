package youtube

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// browserUserAgent is used for HTML pages and direct media requests. It is a
// normal desktop Chrome identifier, not a mobile InnerTube client UA.
const browserUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"

var cookieFilePath string

// SetCookieFile stores the path to a Netscape-format cookie file. The jar is
// built lazily by clientWithCookies so default consent cookies are always
// present.
func SetCookieFile(path string) error {
	cookieFilePath = strings.TrimSpace(path)
	if cookieFilePath == "" {
		return nil
	}
	f, err := os.Open(cookieFilePath)
	if err != nil {
		cookieFilePath = ""
		return err
	}
	f.Close()
	return nil
}

// buildCookieJar creates a jar containing the default YouTube consent cookies
// plus any cookies from the configured file.
func buildCookieJar() *cookiejar.Jar {
	jar, err := newCookieJar(cookieFilePath)
	if err != nil {
		return nil
	}
	return jar
}

// newCookieJar is buildCookieJar with an explicit file path so it can be tested
// without package-global state.
func newCookieJar(path string) (*cookiejar.Jar, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	// Default consent cookies help avoid the "Before you continue" / sorry
	// interstitial for many regions.
	for _, c := range []*http.Cookie{
		{Name: "CONSENT", Value: "YES+cb.20210328-17-p0.en+FX+123", Domain: ".youtube.com", Path: "/"},
		{Name: "SOCS", Value: "CAESEwgDEgk2MTg5NDc4NTYaAmVuIAEaBgiA_LyaBg", Domain: ".youtube.com", Path: "/"},
	} {
		u := &url.URL{Scheme: "https", Host: "www.youtube.com"}
		jar.SetCookies(u, []*http.Cookie{c})
	}
	if path == "" {
		return jar, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return jar, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 7 {
			continue
		}
		domain := fields[0]
		httpOnly := false
		if strings.HasPrefix(domain, "#HttpOnly_") {
			httpOnly = true
			domain = strings.TrimPrefix(domain, "#HttpOnly_")
		}
		if domain == "" {
			continue
		}
		pathField := fields[2]
		if pathField == "" {
			pathField = "/"
		}
		secure := strings.EqualFold(fields[3], "TRUE")
		expiry, _ := strconv.ParseInt(fields[4], 10, 64)
		name := fields[5]
		value := fields[6]
		if name == "" {
			continue
		}
		host := strings.TrimPrefix(domain, ".")
		if host == "" {
			continue
		}
		u := &url.URL{Scheme: "https", Host: host}
		cookie := &http.Cookie{
			Name:     name,
			Value:    value,
			Domain:   domain,
			Path:     pathField,
			Secure:   secure,
			HttpOnly: httpOnly,
		}
		if expiry > 0 {
			cookie.Expires = time.Unix(expiry, 0)
		}
		jar.SetCookies(u, []*http.Cookie{cookie})
	}
	return jar, scanner.Err()
}

// clientWithCookies returns an HTTP client carrying the cookie jar.
func clientWithCookies() *http.Client {
	c := &http.Client{Timeout: 2 * time.Minute}
	c.Jar = buildCookieJar()
	// Header injection to look more like a real browser for HTML pages.
	c.Transport = &headerTransport{base: http.DefaultTransport}
	return c
}

// headerTransport injects browser-like headers.
type headerTransport struct {
	base http.RoundTripper
}

func (h *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := h.base
	if base == nil {
		base = http.DefaultTransport
	}
	// Do not override an explicit User-Agent (kkdai sets its client UA).
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", browserUserAgent)
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	resp, err := base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	// If YouTube redirects to the sorry page, surface a clear error instead of
	// silently following it.
	if resp != nil && resp.Request != nil && isSorryURL(resp.Request.URL) {
		resp.Body.Close()
		return nil, fmt.Errorf("youtube bot doğrulaması (sorry) tetiklendi; --cookies ile oturum açın")
	}
	return resp, nil
}

// isSorryURL reports whether u is Google's "unusual traffic" interstitial.
func isSorryURL(u *url.URL) bool {
	if u == nil {
		return false
	}
	return strings.Contains(u.Host, "google.com") && strings.Contains(u.Path, "/sorry")
}
