package download

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"MusicLeCLI/internal/td/logging"
	"MusicLeCLI/internal/td/model"
	"MusicLeCLI/internal/td/spotify"
	"MusicLeCLI/internal/td/util"
	"MusicLeCLI/internal/td/youtube"
)

// extFromContainer maps a media container to the file extension used on disk.
func extFromContainer(c string) string {
	switch c {
	case "m4a", "mp4":
		return "m4a"
	case "opus":
		return "opus"
	case "ogg":
		return "ogg"
	case "webm":
		return "webm"
	default:
		return "webm"
	}
}

var unsafeFileName = regexp.MustCompile(`\s+`)

// OutputTemplate, when non-empty, overrides BuildFileName's default naming.
// Supported placeholders: {artist}, {title}, {id}, {ext}.
var OutputTemplate string

// SetOutputTemplate sets the file name template used by BuildFileName.
func SetOutputTemplate(t string) { OutputTemplate = t }

// BuildFileName creates a human-readable file name: "Artist - Title [videoID].ext".
func BuildFileName(artist, title, videoID, container string) string {
	if OutputTemplate != "" {
		ext := extFromContainer(container)
		base := OutputTemplate
		base = strings.ReplaceAll(base, "{artist}", strings.TrimSpace(artist))
		base = strings.ReplaceAll(base, "{title}", strings.TrimSpace(title))
		base = strings.ReplaceAll(base, "{id}", videoID)
		base = strings.ReplaceAll(base, "{ext}", ext)
		base = util.SanitizeName(base)
		base = unsafeFileName.ReplaceAllString(base, " ")
		base = strings.TrimSpace(base)
		if base == "" {
			base = videoID
		}
		if !strings.HasSuffix(strings.ToLower(base), "."+ext) {
			base += "." + ext
		}
		return base
	}
	base := strings.TrimSpace(title)
	if base == "" {
		base = videoID
	}
	// Do not repeat the artist when the title already starts with it
	// (very common on YouTube: "Rick Astley - Never Gonna Give You Up").
	if a := strings.TrimSpace(artist); a != "" && !strings.EqualFold(a, "unknown") && !startsWithArtist(base, a) {
		base = a + " - " + base
	}
	base = util.SanitizeName(base)
	base = unsafeFileName.ReplaceAllString(base, " ")
	base = strings.TrimSpace(base)
	// Keep the final name under common filesystem limits (255 bytes) with room
	// for the " [id].ext" suffix and multi-byte UTF-8 characters.
	for len([]rune(base)) > 100 || len(base) > 180 {
		r := []rune(base)
		if len(r) <= 1 {
			break
		}
		base = string(r[:len(r)-1])
	}
	base = strings.TrimSpace(base)
	if base == "" {
		base = videoID
	}
	if videoID != "" {
		base = base + " [" + videoID + "]"
	}
	return base + "." + extFromContainer(container)
}

// DownloadYouTube downloads a YouTube video's best audio to outputDir.
//
// A URL or an 11-character video id is used verbatim: no search is performed,
// so the exact video that was linked is the one that gets downloaded. Only a
// free-text input triggers a search, and even then the best match is chosen by
// scoring title/channel instead of blindly taking the first result.
func DownloadYouTube(ctx context.Context, input, outputDir, playlist string, trackNum int, progress func(float64)) (*model.TrackInfo, string, error) {
	target := strings.TrimSpace(input)
	logging.Printf("youtube download input=%q output=%q playlist=%q", target, outputDir, playlist)
	if !youtube.LooksLikeURL(target) {
		results, err := youtube.SearchYouTubeContext(ctx, target, 8)
		if err != nil {
			return nil, "", fmt.Errorf("youtube search: %w", err)
		}
		best := youtube.PickBest(results, target, "", 0)
		if best == nil {
			return nil, "", fmt.Errorf("no usable result for %q", target)
		}
		target = best.URL()
	}
	v, err := youtube.VideoInfo(target)
	if err != nil {
		logging.Printf("youtube video info failed input=%q: %v", target, err)
		return nil, "", fmt.Errorf("video info: %w", err)
	}
	ti, filePath, err := downloadVideo(ctx, v, outputDir, playlist, trackNum, progress)
	if err != nil {
		logging.Printf("youtube download failed id=%s: %v", v.ID, err)
		return nil, "", err
	}
	return ti, convertIfRequested(ctx, ti, filePath, progress), nil
}

// convertIfRequested applies the optional MP3 conversion and warns (without
// failing the download) when the source cannot be converted.
func convertIfRequested(ctx context.Context, ti *model.TrackInfo, filePath string, progress func(float64)) string {
	out, err := maybeConvertToMP3(ctx, ti, filePath, progress)
	if err != nil {
		// The file on disk stayed in its source container. Record that actual
		// extension so song_list.txt and the printed format do not claim mp4
		// while the file is .m4a.
		if ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(out)), "."); ext != "" {
			ti.Format = ext
		}
		fmt.Fprintf(os.Stderr, "  uyari: %v (dosya %s olarak kaydedildi)\n", err, filepath.Ext(out))
	}
	return out
}

// ArtistTitle renders "Artist - Title" for display, without repeating the
// artist when the title already begins with it.
func ArtistTitle(artist, title string) string {
	base := strings.TrimSpace(title)
	a := strings.TrimSpace(artist)
	if a == "" || strings.EqualFold(a, "unknown") || startsWithArtist(base, a) {
		return base
	}
	return a + " - " + base
}

// startsWithArtist reports whether title already begins with the artist name,
// ignoring accents, case and punctuation.
func startsWithArtist(title, artist string) bool {
	t := normalizeForCompare(title)
	a := normalizeForCompare(artist)
	if a == "" {
		return false
	}
	return strings.HasPrefix(t, a)
}

// normalizeForCompare lowercases and strips everything but letters and digits.
func normalizeForCompare(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r > 127 {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// downloadVideo performs the actual format selection and transfer.
//
// Every viable audio format is tried in order: YouTube hands out URLs that can
// expire or be rejected with 403 for a specific itag, so a failed transfer is
// retried with the next best stream instead of aborting the whole download.
func downloadVideo(ctx context.Context, v *youtube.Video, outputDir, playlist string, trackNum int, progress func(float64)) (*model.TrackInfo, string, error) {
	formats := youtube.AudioFormats(v)
	if len(formats) == 0 {
		return nil, "", fmt.Errorf("video %s: no audio format available", v.ID)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, "", err
	}

	// Build the candidate list, best first. In MP3 mode only formats the pure-Go
	// encoder can handle survive: Opus/WebM first (cheap, audio-only), then
	// AAC/MP4. The combined A/V stream (itag 18) comes last because it costs the
	// whole video's bandwidth.
	candidates := make([]*youtube.Format, 0, len(formats))
	skipped := 0
	for _, f := range formats {
		if MP3Output() && !youtube.MP3Convertible(f) {
			skipped++
			continue
		}
		candidates = append(candidates, f)
	}
	if len(candidates) == 0 {
		if skipped > 0 {
			return nil, "", fmt.Errorf("bu video yalnizca MP3'e cevrilemeyen bir ses formatinda (HE-AAC) mevcut; kaynak formatta indirmek icin --mp3=false kullan")
		}
		return nil, "", fmt.Errorf("bu video icin kullanilabilir ses formati yok")
	}

	// YouTube gates the audio-only itags behind a PO token while still serving
	// the combined A/V stream. Once one audio-only attempt comes back gated,
	// the remaining ones will fail the same way, so jump straight to the
	// combined stream instead of burning four InnerTube clients per itag.
	combinedAt := -1
	for i, f := range candidates {
		if youtube.IsCombinedAV(f) {
			combinedAt = i
			break
		}
	}

	const maxAttempts = 5
	attempts := 0
	var lastErr error
	for i := 0; i < len(candidates) && attempts < maxAttempts; i++ {
		if err := ctx.Err(); err != nil {
			return nil, "", err
		}
		f := candidates[i]
		attempts++
		ti := youtube.TrackInfoFromVideo(v, f, playlist, trackNum)
		container := ti.Format
		if container == "" {
			container = "webm"
			ti.Format = container
		}
		filename := BuildFileName(ti.Artist, ti.Title, v.ID, container)
		destPath := filepath.Join(outputDir, filename)
		// The itag is part of the partial file name: a resume must never append
		// bytes from a different stream that happens to share the container.
		partPath := fmt.Sprintf("%s.%d.part", destPath, f.ItagNo)

		if youtube.IsCombinedAV(f) {
			fmt.Fprintf(os.Stderr, "  bilgi: ses akisi verilmedi, video+ses birlikte indirilip sesi ayiklanacak (%s)\n", strings.ToUpper(container))
		}
		logging.Printf("youtube format attempt=%d itag=%d mime=%s container=%s dest=%q", attempts, f.ItagNo, f.MimeType, container, destPath)
		err := youtube.DownloadAudio(ctx, v, f, partPath, scaleProgress(progress, 0, EncodeProgressStart))
		if err == nil {
			if rerr := os.Rename(partPath, destPath); rerr != nil {
				_ = os.Remove(partPath)
				return nil, "", rerr
			}
			// A successful download makes every other partial for this file
			// obsolete (for example a gated stream tried earlier).
			removeStaleParts(destPath)
			ti.StreamURL = fmt.Sprintf("https://www.youtube.com/watch?v=%s", v.ID)
			ti.Source = v.ID
			// The on-disk extension is authoritative (a combined A/V mp4 is
			// saved as .m4a by extFromContainer).
			ti.Format = strings.TrimPrefix(extFromContainer(container), ".")
			logging.Printf("youtube download ok id=%s itag=%d file=%q", v.ID, f.ItagNo, destPath)
			return ti, destPath, nil
		}
		lastErr = err
		// Keep a non-empty partial for a later resume; drop an empty one.
		if st, serr := os.Stat(partPath); serr == nil && st.Size() == 0 {
			_ = os.Remove(partPath)
		}
		logging.Printf("youtube format failed id=%s itag=%d: %v", v.ID, f.ItagNo, err)

		if combinedAt > i && isGatedError(err) {
			i = combinedAt - 1 // the loop's i++ lands on the combined stream
		}
	}
	return nil, "", describeDownloadFailure(attempts, lastErr)
}

// removeStaleParts deletes "<file>.<itag>.part" leftovers after a successful
// download. The glob is anchored on the exact destination name.
func removeStaleParts(destPath string) {
	matches, err := filepath.Glob(destPath + ".*.part")
	if err != nil {
		return
	}
	for _, m := range matches {
		_ = os.Remove(m)
	}
}

// isGatedError reports whether a transfer failure looks like YouTube's
// PO-token / bot attestation gate rather than a transient network problem.
func isGatedError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "403") ||
		strings.Contains(msg, "cipher not found") ||
		strings.Contains(msg, "signature tokens") ||
		strings.Contains(msg, "sign in") ||
		strings.Contains(msg, "not a bot")
}

// describeDownloadFailure turns a raw transfer error into a message that says
// what actually went wrong. Since 2025 YouTube gates many videos behind a PO
// token / bot attestation: the metadata call succeeds and only the media URLs
// answer 403 (or the signature cannot be deciphered), so the plain "cipher not
// found" text is misleading.
func describeDownloadFailure(attempts int, err error) error {
	if err == nil {
		return fmt.Errorf("indirme basarisiz (%d format denendi)", attempts)
	}
	if isGatedError(err) {
		return fmt.Errorf("indirme basarisiz (%d format denendi): YouTube bu video icin PO token / bot dogrulamasi istiyor, saf Go istemciyle ses akisi alinamiyor (son hata: %v)", attempts, youtube.FriendlyError(err))
	}
	return fmt.Errorf("indirme basarisiz (%d format denendi): %w", attempts, youtube.FriendlyError(err))
}

// YouTubePlaylistItem is one downloaded playlist entry.
type YouTubePlaylistItem struct {
	Track *model.TrackInfo
	File  string
	Err   error
}

// DownloadYouTubePlaylist downloads every playable video in a playlist.
func DownloadYouTubePlaylist(ctx context.Context, input, outputDir, playlistName string, workers int, progress func(int, int, string, float64)) ([]YouTubePlaylistItem, string, error) {
	logging.Printf("youtube playlist start input=%q output=%q workers=%d", input, outputDir, workers)
	pl, err := youtube.PlaylistInfo(input)
	if err != nil {
		return nil, "", fmt.Errorf("playlist info: %w", err)
	}
	name := playlistName
	if name == "" {
		name = pl.Title
	}
	if workers < 1 {
		workers = 1
	}
	if workers > 4 {
		workers = 4
	}
	total := len(pl.Videos)
	out := make([]YouTubePlaylistItem, total)
	jobs := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				entry := pl.Videos[i]
				if progress != nil {
					progress(i+1, total, entry.Title, 0)
				}
				v, err := youtube.VideoInfo("https://www.youtube.com/watch?v=" + entry.ID)
				if err != nil {
					out[i] = YouTubePlaylistItem{Err: err}
					if progress != nil {
						progress(i+1, total, entry.Title, 100)
					}
					continue
				}
				itemProgress := func(p float64) {
					if progress != nil {
						progress(i+1, total, entry.Title, p)
					}
				}
				ti, file, err := downloadVideo(ctx, v, outputDir, name, i+1, itemProgress)
				if err == nil {
					file = convertIfRequested(ctx, ti, file, itemProgress)
				}
				out[i] = YouTubePlaylistItem{Track: ti, File: file, Err: err}
				if progress != nil {
					progress(i+1, total, entry.Title, 100)
				}
			}
		}()
	}
	for i := 0; i < total; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	okCount := 0
	for _, it := range out {
		if it.Err == nil && it.Track != nil {
			okCount++
		}
	}
	logging.Printf("youtube playlist done name=%q downloaded=%d/%d", name, okCount, total)
	return out, name, nil
}

// ResolveSpotifyTrack resolves a Spotify URL/URI/ID or free search query into a
// Spotify track (when possible) and a YouTube search query.
func ResolveSpotifyTrack(ctx context.Context, client *spotify.Client, input string) (*spotify.FullTrack, string, error) {
	trimmed := strings.TrimSpace(input)
	if client == nil {
		return nil, trimmed, nil
	}
	kind, id := spotify.Classify(trimmed)
	if kind == spotify.KindUnknown && spotify.LooksLikeID(trimmed) {
		if resolved, resolvedID, err := client.ResolveAny(ctx, trimmed); err == nil {
			kind, id = resolved, resolvedID
		}
	}
	if kind == spotify.KindTrack && id != "" {
		if tr, err := client.GetTrack(ctx, id); err == nil {
			return tr, spotify.SearchQuery(tr), nil
		}
	}
	return nil, trimmed, nil
}

// DownloadSpotify downloads a Spotify track (or a search query) via YouTube.
// Pass a non-empty ytOverride to pin the exact YouTube source and skip the
// matchmaking search entirely.
func DownloadSpotify(ctx context.Context, client *spotify.Client, input, outputDir, playlist string, trackNum int, progress func(float64)) (*model.TrackInfo, string, error) {
	return DownloadSpotifyURL(ctx, client, input, "", outputDir, playlist, trackNum, progress)
}

// DownloadSpotifyURL is DownloadSpotify with an optional fixed YouTube source.
func DownloadSpotifyURL(ctx context.Context, client *spotify.Client, input, ytOverride, outputDir, playlist string, trackNum int, progress func(float64)) (*model.TrackInfo, string, error) {
	logging.Printf("spotify download input=%q yt=%q output=%q playlist=%q", input, ytOverride, outputDir, playlist)
	tr, query, err := ResolveSpotifyTrack(ctx, client, input)
	if err != nil {
		return nil, "", err
	}
	ytURL := strings.TrimSpace(ytOverride)
	if ytURL != "" {
		if !youtube.LooksLikeURL(ytURL) {
			found, berr := youtube.SearchYouTubeContext(ctx, ytURL, 1)
			if berr != nil || len(found) == 0 {
				return nil, "", fmt.Errorf("gecersiz --yt kaynagi: %q", ytOverride)
			}
			ytURL = found[0].URL()
		}
	} else {
		results, serr := youtube.SearchYouTubeContext(ctx, query, 10)
		if serr != nil {
			return nil, "", fmt.Errorf("youtube search: %w", serr)
		}
		wantTitle, wantArtist, wantDur := query, "", 0
		if tr != nil {
			wantTitle, wantArtist = tr.Name, stiArtist(tr)
			wantDur = int(time.Duration(tr.Duration) * time.Millisecond / time.Second)
		}
		best := youtube.PickBest(results, wantTitle, wantArtist, wantDur)
		if best == nil {
			return nil, "", fmt.Errorf("no usable youtube match for %q", query)
		}
		ytURL = best.URL()
	}
	v, err := youtube.VideoInfo(ytURL)
	if err != nil {
		return nil, "", fmt.Errorf("youtube info: %w", err)
	}
	ti, filePath, err := downloadVideo(ctx, v, outputDir, playlist, trackNum, progress)
	if err != nil {
		return nil, "", err
	}
	if tr != nil {
		sti := spotify.TrackInfoFromFullTrack(tr, playlist, trackNum)
		ti.Title = sti.Title
		ti.Artist = sti.Artist
		ti.Album = sti.Album
		ti.DurationSec = sti.DurationSec
		if sti.Thumbnail != "" {
			ti.Thumbnail = sti.Thumbnail
		}
		ti.Source = sti.Source + "|yt:" + v.ID
	} else {
		ti.Source = "search:" + query + "|yt:" + v.ID
	}
	ti.StreamURL = ytURL
	// Rename the file to reflect Spotify's canonical artist/title.
	want := BuildFileName(stiArtist(tr), bizTitle(tr, ti.Title), v.ID, ti.Format)
	if want != filepath.Base(filePath) {
		newPath := filepath.Join(filepath.Dir(filePath), want)
		if _, statErr := os.Stat(newPath); os.IsNotExist(statErr) {
			if err := os.Rename(filePath, newPath); err == nil {
				filePath = newPath
			}
		} else {
			// Already downloaded: keep the existing file, drop the duplicate.
			_ = os.Remove(filePath)
			filePath = newPath
		}
	}
	filePath = convertIfRequested(ctx, ti, filePath, progress)
	logging.Printf("spotify download ok title=%q artist=%q file=%q", ti.Title, ti.Artist, filePath)
	return ti, filePath, nil
}

func stiArtist(tr *spotify.FullTrack) string {
	if tr == nil || len(tr.Artists) == 0 {
		return ""
	}
	return tr.Artists[0].Name
}

func bizTitle(tr *spotify.FullTrack, fallback string) string {
	if tr == nil || tr.Name == "" {
		return fallback
	}
	return tr.Name
}
