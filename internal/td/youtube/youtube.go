package youtube

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"MusicLe/internal/td/logging"
	"MusicLe/internal/td/model"
	"github.com/kkdai/youtube/v2"
)

// clientSequence is the order in which InnerTube clients are tried. YouTube
// regularly breaks one client while another keeps working, so every operation
// falls through this list instead of depending on a single hard-coded client.
//
// The ANDROID client comes first: it is currently the only one that returns
// playable stream URLs for a wide range of videos (ANDROID_VR, which is the
// library default, hands out URLs that answer 403).
var clientSequence = []youtube.ClientInfo{
	youtube.AndroidClient,
	youtube.AndroidVRClient,
	youtube.IOSClient,
	youtube.WebClient,
	youtube.EmbeddedClient,
}

// ytMu serialises access to the library's package-level client info.
//
// kkdai/youtube stores client info as a pointer to the global DefaultClient
// (c.client = &DefaultClient), so two clients cannot hold different client
// infos at the same time: every client object sees whatever the global holds
// right now. Each attempt therefore takes the lock, pins the global for its
// whole duration, and only then touches the network. Attempts are atomic;
// playlist downloads queue behind each other instead of corrupting each other.
var ytMu sync.Mutex

// newClient returns a client with a sane HTTP timeout and the loaded cookies.
func newClient() *youtube.Client {
	return &youtube.Client{HTTPClient: clientWithCookies()}
}

// Video is a YouTube video (alias of the upstream type).
type Video = youtube.Video

// Format is a YouTube stream format (alias of the upstream type).
type Format = youtube.Format

// Thumbnails is a list of YouTube thumbnails (alias of the upstream type).
type Thumbnails = youtube.Thumbnails

// VideoInfo fetches metadata for a YouTube video URL or ID, trying every
// known InnerTube client until one answers.
func VideoInfo(input string) (*youtube.Video, error) {
	return VideoInfoContext(context.Background(), input)
}

// VideoInfoContext is VideoInfo with cancellation support.
func VideoInfoContext(ctx context.Context, input string) (*youtube.Video, error) {
	var lastErr error
	for _, info := range clientSequence {
		v, err := videoInfoWith(info, ctx, input)
		if err == nil && v != nil {
			return v, nil
		}
		if err != nil {
			lastErr = err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("video bulunamadi")
	}
	return nil, localizeError(lastErr)
}

// videoInfoWith fetches a video using one specific InnerTube client.
func videoInfoWith(info youtube.ClientInfo, ctx context.Context, input string) (*youtube.Video, error) {
	ytMu.Lock()
	defer ytMu.Unlock()
	youtube.DefaultClient = info
	return newClient().GetVideoContext(ctx, input)
}

// PlaylistInfo fetches a YouTube playlist, trying every known client. A client
// that returns the playlist but no title (ANDROID does this) is kept only as a
// fallback: a titled response is preferred so the playlist gets a real name.
func PlaylistInfo(input string) (*youtube.Playlist, error) {
	var fallback *youtube.Playlist
	var lastErr error
	for _, info := range clientSequence {
		ytMu.Lock()
		youtube.DefaultClient = info
		p, err := newClient().GetPlaylistContext(context.Background(), input)
		ytMu.Unlock()
		if err == nil && p != nil {
			if strings.TrimSpace(p.Title) != "" {
				return p, nil
			}
			if fallback == nil {
				fallback = p
			}
			continue
		}
		if err != nil {
			lastErr = err
		}
	}
	if fallback != nil {
		return fallback, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("playlist bulunamadi")
	}
	return nil, localizeError(lastErr)
}

// IsPlaylistURL reports whether the input refers to a whole playlist. It is a
// convenience wrapper around Classify.
func IsPlaylistURL(input string) bool {
	return Classify(input).Kind == KindPlaylist
}

// AudioFormats returns every usable audio format, best candidate first:
// audio-only before combined A/V, Opus/WebM before M4A/AAC, then bitrate.
func AudioFormats(v *youtube.Video) []*youtube.Format {
	var candidates []youtube.Format
	for i := range v.Formats {
		f := v.Formats[i]
		if f.AudioChannels <= 0 {
			continue
		}
		candidates = append(candidates, f)
	}
	if len(candidates) == 0 {
		return nil
	}
	score := func(f youtube.Format) int {
		m := strings.ToLower(f.MimeType)
		base := 1
		// Prefer audio-only streams; combined A/V streams waste bandwidth.
		audioOnly := f.Width == 0 && f.Height == 0
		if strings.Contains(m, "opus") {
			base = 3
		} else if strings.Contains(m, "mp4") || strings.Contains(m, "m4a") || strings.Contains(m, "webm") {
			base = 2
		}
		if audioOnly {
			base += 10
		}
		return base
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		si, sj := score(candidates[i]), score(candidates[j])
		if si != sj {
			return si > sj
		}
		if candidates[i].Bitrate != candidates[j].Bitrate {
			return candidates[i].Bitrate > candidates[j].Bitrate
		}
		return candidates[i].AudioSampleRate > candidates[j].AudioSampleRate
	})
	out := make([]*youtube.Format, len(candidates))
	for i := range candidates {
		out[i] = &candidates[i]
	}
	return out
}

// MP3Convertible reports whether a format can be turned into MP3 by the
// pure-Go encoder chain: Opus (WebM) and AAC-LC (MP4/M4A) both qualify.
//
// HE-AAC (mp4a.40.5) is rejected because the decoder has no SBR/PS support, and
// so is every non-AAC codec.
func MP3Convertible(f *youtube.Format) bool {
	if f == nil || f.AudioChannels <= 0 {
		return false
	}
	m := strings.ToLower(f.MimeType)
	switch {
	case strings.Contains(m, "opus"):
		return true
	case strings.Contains(m, "mp4a.40.5"):
		return false // HE-AAC (SBR), unsupported
	case strings.Contains(m, "mp4a"):
		return true // AAC-LC
	}
	return false
}

// IsCombinedAV reports whether the format carries video alongside its audio.
// Those streams are the fallback when YouTube withholds the audio-only ones,
// but they cost the whole video's bandwidth.
func IsCombinedAV(f *youtube.Format) bool {
	return f != nil && (f.Width != 0 || f.Height != 0)
}

// ChooseAudioFormat picks the best audio format, or an error when none exists.
func ChooseAudioFormat(v *youtube.Video) (*youtube.Format, error) {
	formats := AudioFormats(v)
	if len(formats) == 0 {
		return nil, fmt.Errorf("no audio format available")
	}
	return formats[0], nil
}

// DownloadAudio downloads the chosen format to destPath, reporting progress
// (0-100). If the stream fails mid-transfer it is retried with the remaining
// clients, since a rejected URL is a client problem rather than a bad video.
func DownloadAudio(ctx context.Context, v *youtube.Video, f *youtube.Format, destPath string, progress func(float64)) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	var lastErr error
	for _, info := range clientSequence {
		if err := ctx.Err(); err != nil {
			return err
		}
		// Retry each client a couple of times on transient errors so a single
		// hiccup does not abort the whole download.
		for attempt := 0; attempt < 2; attempt++ {
			if err := ctx.Err(); err != nil {
				return err
			}
			err := downloadAttempt(info, ctx, v.ID, f, destPath, progress)
			if err == nil {
				return nil
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			lastErr = err
			logging.Printf("youtube stream failed client=%s itag=%d attempt=%d: %v", info.Name, f.ItagNo, attempt+1, err)
			// Keep the partial file: the next attempt (or the next run) resumes
			// it with an HTTP Range request instead of starting over.
			if attempt == 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(retryDelay(err)):
				}
			}
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("stream indirilemedi")
	}
	return lastErr
}

// retryDelay returns a jittered backoff. Rate-limit errors get a longer base
// delay so a playlist does not hammer YouTube while it is throttling us.
func retryDelay(err error) time.Duration {
	base := 750 * time.Millisecond
	if isRateLimitError(err) {
		base = 2500 * time.Millisecond
	}
	// +/- 25% jitter keeps parallel workers from retrying in lockstep.
	jitter := time.Duration(rand.Int63n(int64(base)/2)) - base/4
	d := base + jitter
	if d < 100*time.Millisecond {
		d = 100 * time.Millisecond
	}
	return d
}

// isRateLimitError reports whether the upstream error looks like throttling.
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "rate limited") ||
		strings.Contains(msg, "too many requests") ||
		strings.Contains(msg, "429")
}

// downloadAttempt fetches the video again with one client (stream URLs are
// bound to the client that produced the player response) and transfers it
// while holding the client lock.
func downloadAttempt(info youtube.ClientInfo, ctx context.Context, videoID string, wanted *youtube.Format, destPath string, progress func(float64)) error {
	ytMu.Lock()
	defer ytMu.Unlock()
	youtube.DefaultClient = info

	c := newClient()
	video, err := c.GetVideoContext(ctx, "https://www.youtube.com/watch?v="+videoID)
	if err != nil || video == nil {
		if err != nil {
			return err
		}
		return fmt.Errorf("video bilgisi alinamadi: %s", videoID)
	}
	format := pickEquivalentFormat(video, wanted)
	if format == nil {
		return fmt.Errorf("format %d (%s) yok", wanted.ItagNo, wanted.MimeType)
	}
	return downloadWithClient(ctx, c, video, format, destPath, progress)
}

// pickEquivalentFormat finds the same itag (or the closest alternative) in a
// freshly fetched video description.
func pickEquivalentFormat(video *youtube.Video, wanted *youtube.Format) *youtube.Format {
	if video == nil || wanted == nil {
		return nil
	}
	for i := range video.Formats {
		if video.Formats[i].ItagNo == wanted.ItagNo {
			f := video.Formats[i]
			return &f
		}
	}
	if f, err := ChooseAudioFormat(video); err == nil {
		return f
	}
	return nil
}

// downloadWithClient performs one transfer attempt with a single client. The
// stream URL is resolved first (this is where signature deciphering happens)
// and the bytes are then downloaded with Range support so an interrupted
// transfer can resume.
func downloadWithClient(ctx context.Context, c *youtube.Client, v *youtube.Video, f *youtube.Format, destPath string, progress func(float64)) error {
	streamURL, err := c.GetStreamURLContext(ctx, v, f)
	if err != nil {
		return err
	}
	return downloadStream(ctx, c.HTTPClient, streamURL, f.ContentLength, destPath, progress)
}

// TrackInfoFromVideo builds a TrackInfo from a YouTube video and chosen format.
func TrackInfoFromVideo(v *youtube.Video, f *youtube.Format, playlist string, num int) *model.TrackInfo {
	ti := &model.TrackInfo{
		Title:       v.Title,
		Artist:      v.Author,
		Album:       "",
		DurationSec: int(v.Duration.Seconds()),
		StreamURL:   fmt.Sprintf("https://www.youtube.com/watch?v=%s", v.ID),
		Format:      ContainerForFormat(f),
		ContentLen:  f.ContentLength,
		Thumbnail:   "",
		Playlist:    playlist,
		TrackNum:    num,
		Source:      v.ID,
	}
	if len(v.Thumbnails) > 0 {
		ti.Thumbnail = v.Thumbnails[0].URL
	}
	if ti.Format == "" {
		ti.Format = "webm"
	}
	return ti
}

// ContainerForFormat maps a stream format to a file extension. Combined A/V
// streams are a real MP4 file, not an audio-only M4A, so they get their own
// extension; the demuxer accepts either.
func ContainerForFormat(f *youtube.Format) string {
	if f == nil {
		return "webm"
	}
	name := ContainerFromMime(f.MimeType)
	if IsCombinedAV(f) && strings.Contains(strings.ToLower(f.MimeType), "mp4") {
		return "mp4"
	}
	return name
}

// ContainerFromMime maps a mime type to a container name.
func ContainerFromMime(mime string) string {
	m := strings.ToLower(mime)
	switch {
	case strings.Contains(m, "webm"):
		return "webm"
	case strings.Contains(m, "mp4"), strings.Contains(m, "m4a"):
		return "m4a"
	case strings.Contains(m, "ogg"):
		return "ogg"
	case strings.Contains(m, "opus"):
		return "opus"
	default:
		// strip parameters
		if i := strings.Index(m, ";"); i >= 0 {
			m = m[:i]
		}
		if i := strings.Index(m, "/"); i >= 0 {
			m = m[i+1:]
		}
		if m == "" {
			return "webm"
		}
		return m
	}
}
