package youtube

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// downloadStream transfers streamURL into destPath. If destPath already holds a
// partial transfer it is resumed with an HTTP Range request; servers that
// ignore Range are detected and the file is restarted from zero.
//
// total is the expected size (0 when unknown). On a mid-transfer failure the
// partial file is deliberately kept so the next invocation can resume it.
func downloadStream(ctx context.Context, hc *http.Client, streamURL string, total int64, destPath string, progress func(float64)) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}
	if hc == nil {
		hc = http.DefaultClient
	}

	var existing int64
	if st, err := os.Stat(destPath); err == nil {
		existing = st.Size()
	}
	if total > 0 && existing == total {
		if progress != nil {
			progress(100)
		}
		return nil
	}
	if total > 0 && existing > total {
		// A stale partial from another stream/format: start over.
		existing = 0
		_ = os.Remove(destPath)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	if existing > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", existing))
	}

	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	start := existing
	switch resp.StatusCode {
	case http.StatusPartialContent:
		crStart, crTotal, ok := parseContentRange(resp.Header.Get("Content-Range"))
		if existing > 0 {
			if !ok || crStart != existing {
				// The partial file does not line up with what the server is
				// willing to send; discard it so a later run starts clean.
				_ = os.Remove(destPath)
				return fmt.Errorf("beklenmeyen Content-Range: %q", resp.Header.Get("Content-Range"))
			}
		} else {
			start = 0
		}
		if total <= 0 && crTotal > 0 {
			total = crTotal
		}
	case http.StatusOK:
		// The server ignored Range and sent the whole stream; rewrite the file.
		start = 0
		if total <= 0 {
			total = resp.ContentLength
		}
	case http.StatusRequestedRangeNotSatisfiable:
		if total > 0 && existing == total {
			if progress != nil {
				progress(100)
			}
			return nil
		}
		_ = os.Remove(destPath)
		return fmt.Errorf("range istegi reddedildi (mevcut %d byte)", existing)
	default:
		return fmt.Errorf("akis status %d", resp.StatusCode)
	}

	flags := os.O_CREATE | os.O_WRONLY
	if start > 0 {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	out, err := os.OpenFile(destPath, flags, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	if progress != nil && total > 0 && start > 0 {
		progress(float64(start) / float64(total) * 100.0)
	}

	written := start
	buf := make([]byte, 32*1024)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return werr
			}
			written += int64(n)
			if progress != nil && total > 0 {
				p := float64(written) / float64(total) * 100.0
				if p > 100 {
					p = 100
				}
				progress(p)
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return rerr
		}
	}

	if written == 0 {
		return fmt.Errorf("bos akis")
	}
	if total > 0 && written != total {
		return fmt.Errorf("eksik indirme: %d/%d byte", written, total)
	}
	if progress != nil {
		progress(100)
	}
	return nil
}

// parseContentRange parses a "bytes <start>-<end>/<total>" header value.
// total is 0 when the server reports "*" or omits it.
func parseContentRange(v string) (start, total int64, ok bool) {
	v = strings.TrimSpace(v)
	if !strings.HasPrefix(v, "bytes ") {
		return 0, 0, false
	}
	v = strings.TrimPrefix(v, "bytes ")
	parts := strings.SplitN(v, "/", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	dash := strings.IndexByte(parts[0], '-')
	if dash <= 0 {
		return 0, 0, false
	}
	parsedStart, err := strconv.ParseInt(parts[0][:dash], 10, 64)
	if err != nil {
		return 0, 0, false
	}
	if parts[1] != "*" {
		total, _ = strconv.ParseInt(parts[1], 10, 64)
	}
	return parsedStart, total, true
}
