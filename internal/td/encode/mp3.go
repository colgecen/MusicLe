// Package encode converts downloaded audio into other containers/codecs using
// pure Go codecs only. No external binaries (ffmpeg, lame, ...) are involved.
package encode

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/at-wat/ebml-go/mkvcore"
	"github.com/bogem/id3v2/v2"
	"github.com/braheezy/shine-mp3/pkg/mp3"
	"github.com/pion/opus"

	"MusicLe/internal/td/mp4audio"
)

const (
	// opusSampleRate is the fixed Opus output rate. Opus always decodes at
	// 48 kHz, so no resampling is needed for that path.
	opusSampleRate = 48000
	// mp3Channels is the fixed output channel count. Mono sources are upmixed
	// by the decoders themselves.
	mp3Channels = 2
	// opusMaxFrameSamples is the longest Opus frame (120 ms at 48 kHz).
	opusMaxFrameSamples = 5760
	// mp3FrameSamples is one MPEG-1 Layer III frame, in samples per channel.
	mp3FrameSamples = 1152
)

// mp3BitrateKbps is the shine-mp3 encoding bitrate. Defaults to 128 and can
// be changed via SetMP3Bitrate before encoding.
var mp3BitrateKbps = 128

// SetMP3Bitrate changes the MP3 encoding bitrate (kbps). Supported values are
// the standard MPEG-1 Layer III bitrates.
func SetMP3Bitrate(kbps int) error {
	switch kbps {
	case 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320:
		mp3BitrateKbps = kbps
		return nil
	default:
		return fmt.Errorf("desteklenmeyen bitrate: %d (32..320 kbps)", kbps)
	}
}

// bitRatesTable mirrors shine-mp3's MPEG bit-rate table (kbps).
var bitRatesTable = [16][4]int{
	{-1, -1, -1, -1}, {8, -1, 8, 32}, {16, -1, 16, 40}, {24, -1, 24, 48},
	{32, -1, 32, 56}, {40, -1, 40, 64}, {48, -1, 48, 80}, {56, -1, 56, 96},
	{64, -1, 64, 112}, {-1, -1, 80, 128}, {-1, -1, 96, 160}, {-1, -1, 112, 192},
	{-1, -1, 128, 224}, {-1, -1, 144, 256}, {-1, -1, 160, 320}, {-1, -1, -1, -1},
}

// setEncoderBitrate reconfigures a shine encoder for the requested bitrate.
func setEncoderBitrate(enc *mp3.Encoder, bitrate int) {
	mpegVer := int(enc.Mpeg.Version)
	col := 3
	switch mpegVer {
	case 0:
		col = 0
	case 2:
		col = 2
	case 3:
		col = 3
	}
	idx := 9 // 128 kbps default index
	for i := 0; i < 16; i++ {
		if bitRatesTable[i][col] == bitrate {
			idx = i
			break
		}
	}
	enc.Mpeg.Bitrate = int64(bitrate)
	enc.Mpeg.BitrateIndex = int64(idx)
	avg := (float64(enc.Mpeg.GranulesPerFrame) * 1152 / float64(enc.Wave.SampleRate)) * (float64(bitrate) * 1000 / float64(enc.Mpeg.BitsPerSlot))
	enc.Mpeg.WholeSlotsPerFrame = int64(avg)
	enc.Mpeg.FracSlotsPerFrame = avg - float64(enc.Mpeg.WholeSlotsPerFrame)
	enc.Mpeg.SlotLag = -enc.Mpeg.FracSlotsPerFrame
	if enc.Mpeg.FracSlotsPerFrame == 0 {
		enc.Mpeg.Padding = 0
	}
}

// ErrUnsupportedSource means the input is neither WebM/Opus nor MP4/AAC, so the
// pure-Go decoder chain cannot read it.
var ErrUnsupportedSource = errors.New("kaynak MP3'e cevrilemiyor: WebM/Opus veya MP4/AAC dosyasi gerekli")

// Tags are the ID3v2 fields written into the produced MP3.
type Tags struct {
	Title     string
	Artist    string
	Album     string
	Cover     []byte
	CoverMIME string
}

// IsWebM reports whether path starts with the EBML magic used by WebM.
func IsWebM(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	var magic [4]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil {
		return false
	}
	return magic == [4]byte{0x1a, 0x45, 0xdf, 0xa3}
}

// IsSupportedSource reports whether ToMP3 can decode path: WebM/Opus or an
// MP4/M4A container with an AAC-LC track.
func IsSupportedSource(path string) bool {
	if IsWebM(path) {
		return true
	}
	return mp4audio.IsMP4(path)
}

// ToMP3 decodes src (WebM/Opus or MP4/AAC) and re-encodes it as MP3, then
// writes ID3v2 tags. totalSec is the expected length and only drives the
// progress callback; decoding continues until the container ends.
func ToMP3(ctx context.Context, src, dst string, totalSec float64, tags Tags, progress func(float64)) error {
	switch {
	case IsWebM(src):
		return webMToMP3(ctx, src, dst, totalSec, tags, progress)
	case mp4audio.IsMP4(src):
		return aacToMP3(ctx, src, dst, tags, progress)
	default:
		return ErrUnsupportedSource
	}
}

// webMToMP3 is the WebM/Opus half of ToMP3.
func webMToMP3(ctx context.Context, src, dst string, totalSec float64, tags Tags, progress func(float64)) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	readers, err := mkvcore.NewSimpleBlockReader(f)
	if err != nil {
		return fmt.Errorf("webm okunamadi: %w", err)
	}
	for _, r := range readers {
		defer r.Close()
	}
	audio := pickAudioTrack(readers)
	if audio == nil {
		return ErrUnsupportedSource
	}
	if !strings.Contains(strings.ToUpper(audio.TrackEntry().CodecID), "OPUS") {
		return fmt.Errorf("desteklenmeyen ses kodegi: %s", audio.TrackEntry().CodecID)
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	// On any failure the half-written file must not be left behind.
	success := false
	defer func() {
		_ = out.Close()
		if !success {
			_ = os.Remove(dst)
		}
	}()

	decoded, err := encodeOpusPackets(ctx, func() ([]byte, error) {
		packet, _, _, rerr := audio.Read()
		return packet, rerr
	}, out, totalSec, progress)
	if err != nil {
		return err
	}
	if decoded == 0 {
		return fmt.Errorf("ses cozulemedi: bos akis")
	}
	// Close before tagging: writeTags rewrites the file with the ID3 header.
	if err := out.Close(); err != nil {
		return err
	}
	if err := writeTags(dst, tags); err != nil {
		// Tags are a nice-to-have; a valid MP3 without them is still useful.
		_ = err
	}
	success = true
	if progress != nil {
		progress(100)
	}
	return nil
}

// pickAudioTrack returns the first audio track of a WebM file.
func pickAudioTrack(readers []mkvcore.BlockReadCloserWithTrackEntry) mkvcore.BlockReadCloserWithTrackEntry {
	for _, r := range readers {
		entry := r.TrackEntry()
		if entry.TrackType == 2 || strings.HasPrefix(strings.ToUpper(entry.CodecID), "A_") {
			return r
		}
	}
	if len(readers) > 0 {
		return readers[0]
	}
	return nil
}

// encodeOpusPackets decodes Opus packets from next into MP3 frames written to
// out. It is separate from the container handling so it can be tested without
// a real WebM file.
func encodeOpusPackets(ctx context.Context, next func() ([]byte, error), out io.Writer, totalSec float64, progress func(float64)) (int64, error) {
	dec, err := opus.NewDecoderWithOutput(opusSampleRate, mp3Channels)
	if err != nil {
		return 0, err
	}
	w := newMP3Writer(out, opusSampleRate, mp3Channels, totalSec, progress)
	buf := make([]int16, opusMaxFrameSamples*mp3Channels)
	var decoded int64

	for {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		packet, rerr := next()
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return 0, fmt.Errorf("webm blok okunamadi: %w", rerr)
		}
		if len(packet) == 0 {
			continue
		}
		n, derr := dec.DecodeToInt16(packet, buf)
		if derr != nil {
			// A single corrupt packet should not abort the whole file.
			continue
		}
		if n == 0 {
			continue
		}
		decoded += int64(n)
		if err := w.write(buf[:n*mp3Channels]); err != nil {
			return 0, err
		}
	}
	if err := w.close(); err != nil {
		return 0, err
	}
	return decoded, nil
}

// mp3Writer feeds interleaved 16-bit PCM into shine-mp3, cutting it into whole
// MPEG frames and reporting progress.
//
// shine-mp3's Encoder.Write slices interleaved stereo data incorrectly and
// silently drops samples, which stretches the output by ~20%. Feeding whole
// MPEG frames straight to EncodeBufferInterleaved avoids that path entirely.
type mp3Writer struct {
	enc      *mp3.Encoder
	bw       *bufio.Writer
	pending  []int16
	frameLen int
	channels int
	rate     int
	written  int64 // samples per channel
	totalSec float64
	progress func(float64)
	lastPct  float64
	reported bool
}

func newMP3Writer(out io.Writer, sampleRate, channels int, totalSec float64, progress func(float64)) *mp3Writer {
	enc := mp3.NewEncoder(sampleRate, channels)
	setEncoderBitrate(enc, mp3BitrateKbps)
	frameLen := mp3FrameSamples * channels
	return &mp3Writer{
		enc:      enc,
		bw:       bufio.NewWriterSize(out, 1<<16),
		pending:  make([]int16, 0, frameLen*2),
		frameLen: frameLen,
		channels: channels,
		rate:     sampleRate,
		totalSec: totalSec,
		progress: progress,
		lastPct:  -1,
	}
}

// write accepts any number of interleaved samples.
func (w *mp3Writer) write(pcm []int16) error {
	w.pending = append(w.pending, pcm...)
	for len(w.pending) >= w.frameLen {
		if err := w.writeFrame(w.pending[:w.frameLen]); err != nil {
			return err
		}
		w.pending = append(w.pending[:0], w.pending[w.frameLen:]...)
	}
	w.written += int64(len(pcm) / w.channels)
	w.report()
	return nil
}

func (w *mp3Writer) writeFrame(chunk []int16) error {
	data, written := w.enc.EncodeBufferInterleaved(chunk)
	_, err := w.bw.Write(data[:written])
	return err
}

// report throttles progress callbacks to whole percent steps.
func (w *mp3Writer) report() {
	if w.progress == nil || w.totalSec <= 0 || w.rate <= 0 {
		return
	}
	p := float64(w.written) / float64(w.rate) / w.totalSec * 100.0
	if p > 100 {
		p = 100
	}
	if p-w.lastPct >= 1 {
		w.lastPct = p
		w.progress(p)
	}
}

// close flushes the trailing partial frame (padded with silence so the last
// samples are kept) and the buffered writer.
func (w *mp3Writer) close() error {
	if len(w.pending) > 0 {
		tail := make([]int16, w.frameLen)
		copy(tail, w.pending)
		if err := w.writeFrame(tail); err != nil {
			return err
		}
	}
	return w.bw.Flush()
}

// writeTags writes ID3v2.3 text frames (and an optional front cover) into path.
func writeTags(path string, tags Tags) error {
	if tags.Title == "" && tags.Artist == "" && tags.Album == "" && len(tags.Cover) == 0 {
		return nil
	}
	tag, err := id3v2.Open(path, id3v2.Options{Parse: false})
	if err != nil {
		return err
	}
	defer tag.Close()

	tag.SetVersion(4)
	if tags.Title != "" {
		tag.SetTitle(tags.Title)
	}
	if tags.Artist != "" {
		tag.SetArtist(tags.Artist)
	}
	if tags.Album != "" {
		tag.SetAlbum(tags.Album)
	}
	if len(tags.Cover) > 0 {
		mime := tags.CoverMIME
		if mime == "" {
			mime = "image/jpeg"
		}
		tag.AddAttachedPicture(id3v2.PictureFrame{
			Encoding:    id3v2.EncodingUTF8,
			MimeType:    mime,
			PictureType: id3v2.PTFrontCover,
			Description: "Cover",
			Picture:     tags.Cover,
		})
	}
	return tag.Save()
}
