// Package mp4audio extracts the audio track of an MP4 file (the container
// YouTube serves as itag 18) using pure Go, without cgo or external binaries.
//
// It only understands what is needed to feed an AAC decoder: one audio track
// with an mp4a sample entry, its AudioSpecificConfig, and the sample table that
// maps every access unit to a byte range in the file.
package mp4audio

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/abema/go-mp4"
)

// ErrUnsupported means the file is not an MP4 with a single AAC-LC audio track.
var ErrUnsupported = errors.New("mp4: desteklenmeyen dosya (tek AAC-LC ses izi gerekli)")

// ErrHEAAC means the audio track uses HE-AAC (SBR/HE-AACv2). The pure-Go
// decoder has no SBR support, so this file cannot become an MP3.
var ErrHEAAC = errors.New("mp4: ses HE-AAC (SBR); saf Go cozucusu SBR desteklemiyor")

// AAC sample rate table from ISO/IEC 14496-3; index 15 means "explicit rate".
var sampleRates = [...]int{
	96000, 88200, 64000, 48000, 44100, 32000, 24000,
	22050, 16000, 12000, 11025, 8000, 7350,
}

// Track is an open MP4 audio track. Call Close when done.
type Track struct {
	f          *os.File
	asc        []byte
	sampleRate int
	channels   int
	frames     int
	chunks     mp4.Chunks
	samples    mp4.Samples
}

// ASC returns the AudioSpecificConfig of the track.
func (t *Track) ASC() []byte { return t.asc }

// SampleRate is the track's sampling frequency in Hz.
func (t *Track) SampleRate() int { return t.sampleRate }

// Channels is the track's channel count.
func (t *Track) Channels() int { return t.channels }

// Frames is the number of AAC access units in the track.
func (t *Track) Frames() int { return t.frames }

// DurationSec is the track length in seconds, derived from the sample table.
func (t *Track) DurationSec() float64 {
	if t.sampleRate <= 0 {
		return 0
	}
	return float64(t.frames*1024) / float64(t.sampleRate)
}

// Close releases the underlying file.
func (t *Track) Close() error { return t.f.Close() }

// Open inspects path and returns its audio track.
func Open(path string) (*Track, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	t, err := openFile(f)
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	return t, nil
}

func openFile(f *os.File) (*Track, error) {
	info, err := mp4.Probe(f)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnsupported, err)
	}

	var track *mp4.Track
	for _, tr := range info.Tracks {
		if tr.Codec == mp4.CodecMP4A {
			track = tr
			break
		}
	}
	if track == nil {
		return nil, ErrUnsupported
	}
	if len(track.Samples) == 0 || len(track.Chunks) == 0 {
		return nil, ErrUnsupported
	}

	asc, err := readASC(f)
	if err != nil {
		// Some encoders put the config only in the esds descriptor chain; if
		// that fails there is nothing to decode with.
		return nil, fmt.Errorf("%w: %v", ErrUnsupported, err)
	}
	cfg, err := ParseASC(asc)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnsupported, err)
	}
	if cfg.HEAAC {
		return nil, fmt.Errorf("%w (AOT %d)", ErrHEAAC, cfg.ObjectType)
	}
	if !cfg.AACLC {
		return nil, fmt.Errorf("%w: AAC-LC degil (AOT %d)", ErrUnsupported, cfg.ObjectType)
	}
	// Prefer the container's own rate when the ASC leaves it implicit.
	if cfg.SampleRate <= 0 {
		cfg.SampleRate = int(track.Timescale)
	}
	if cfg.SampleRate <= 0 || cfg.Channels <= 0 {
		return nil, ErrUnsupported
	}

	return &Track{
		f:          f,
		asc:        asc,
		sampleRate: cfg.SampleRate,
		channels:   cfg.Channels,
		frames:     len(track.Samples),
		chunks:     track.Chunks,
		samples:    track.Samples,
	}, nil
}

// readASC pulls the DecoderSpecificInfo out of the esds box.
func readASC(f *os.File) ([]byte, error) {
	boxes, err := mp4.ExtractBox(f, nil, mp4.BoxPath{
		mp4.BoxTypeMoov(), mp4.BoxTypeTrak(), mp4.BoxTypeMdia(),
		mp4.BoxTypeMinf(), mp4.BoxTypeStbl(), mp4.BoxTypeStsd(),
		mp4.BoxTypeMp4a(), mp4.BoxTypeEsds(),
	})
	if err != nil {
		return nil, err
	}
	if len(boxes) == 0 {
		return nil, errors.New("esds bulunamadi")
	}
	bi := boxes[0]
	if _, err := bi.SeekToPayload(f); err != nil {
		return nil, err
	}
	payload := make([]byte, bi.Size-bi.HeaderSize)
	if _, err := io.ReadFull(f, payload); err != nil {
		return nil, err
	}
	if len(payload) < 4 {
		return nil, io.ErrUnexpectedEOF
	}
	return descriptorASC(payload[4:]) // skip FullBox version+flags
}

// descriptorASC walks the ES_Descriptor chain in an esds payload and returns
// the DecoderSpecificInfo bytes (tag 0x05).
func descriptorASC(b []byte) ([]byte, error) {
	i := 0
	for i < len(b) {
		tag := b[i]
		i++
		size := 0
		for k := 0; k < 4; k++ {
			if i >= len(b) {
				return nil, io.ErrUnexpectedEOF
			}
			c := b[i]
			i++
			size = size<<7 | int(c&0x7f)
			if c&0x80 == 0 {
				break
			}
		}
		// Bound the declared size by what is actually left: a malformed box
		// must not make us slice past the buffer.
		if size < 0 || i+size > len(b) {
			return nil, io.ErrUnexpectedEOF
		}
		switch tag {
		case 0x03: // ES_Descriptor: ES_ID (2) + flags (1)
			if i+3 > len(b) {
				return nil, io.ErrUnexpectedEOF
			}
			i += 3
		case 0x04: // DecoderConfigDescriptor: OTI + streamType + 3 + 4 + 4
			if i+13 > len(b) {
				return nil, io.ErrUnexpectedEOF
			}
			i += 13
		case 0x05: // DecoderSpecificInfo
			return b[i : i+size], nil
		default:
			i += size
		}
	}
	return nil, errors.New("DecoderSpecificInfo bulunamadi")
}

// Config is a decoded AudioSpecificConfig.
type Config struct {
	ObjectType int
	SampleRate int
	Channels   int
	AACLC      bool
	// HEAAC is true for HE-AAC / HE-AACv2 (SBR or parametric stereo), which
	// needs a decoder extension this project does not have.
	HEAAC bool
	// FreqIndex is the ISO sampling-frequency index, or 15 for an explicit rate.
	FreqIndex int
}

// ParseASC decodes the fields of an AudioSpecificConfig that matter for
// decoding: object type, sample rate and channel configuration.
func ParseASC(asc []byte) (Config, error) {
	var cfg Config
	if len(asc) < 2 {
		return cfg, errors.New("ASC cok kisa")
	}
	br := bitReader{data: asc}
	cfg.ObjectType = int(br.read(5))
	if cfg.ObjectType == 31 {
		cfg.ObjectType = 32 + int(br.read(6))
	}
	cfg.FreqIndex = int(br.read(4))
	if cfg.FreqIndex == 15 {
		cfg.SampleRate = int(br.read(24))
	} else if cfg.FreqIndex < len(sampleRates) {
		cfg.SampleRate = sampleRates[cfg.FreqIndex]
	}
	cfg.Channels = int(br.read(4))
	if br.err {
		return Config{}, errors.New("ASC eksik")
	}
	// Object type 2 is AAC-LC. HE-AAC signals SBR separately (object type 5 or
	// 29), which this decoder chain cannot handle.
	cfg.AACLC = cfg.ObjectType == 2
	cfg.HEAAC = cfg.ObjectType == 5 || cfg.ObjectType == 29
	return cfg, nil
}

// FreqIndexFor returns the ISO sampling-frequency index for a rate, or 15 when
// the rate is not in the table.
func FreqIndexFor(rate int) int {
	for i, r := range sampleRates {
		if r == rate {
			return i
		}
	}
	return 15
}

// ADTSHeader builds a 7-byte ADTS header for a payload of payloadLen bytes.
func ADTSHeader(payloadLen, freqIndex, channels int) []byte {
	frameLen := payloadLen + 7
	h := make([]byte, 7)
	h[0] = 0xFF
	h[1] = 0xF1 // MPEG-4, layer 0, no CRC
	// profile is AOT-1, so AAC-LC (2) becomes 1.
	h[2] = byte((1 << 6) | (freqIndex&0x0F)<<2 | (channels>>2)&0x01)
	h[3] = byte((channels&0x03)<<6 | (frameLen>>11)&0x03)
	h[4] = byte((frameLen >> 3) & 0xFF)
	h[5] = byte((frameLen&0x07)<<5 | 0x1F)
	h[6] = 0xFC
	return h
}

// ForEachFrame calls fn for every access unit, in order. The slice passed to fn
// is reused between calls and must not be retained.
func (t *Track) ForEachFrame(fn func(au []byte) error) error {
	buf := make([]byte, 0, 4096)
	idx := 0
	for _, ch := range t.chunks {
		off := ch.DataOffset
		for k := uint32(0); k < ch.SamplesPerChunk; k++ {
			if idx >= len(t.samples) {
				return nil
			}
			size := int(t.samples[idx].Size)
			if size <= 0 || size > 1<<22 {
				return fmt.Errorf("mp4: gecersiz ornek boyutu %d (frame %d)", size, idx)
			}
			if cap(buf) < size {
				buf = make([]byte, size)
			}
			buf = buf[:size]
			if _, err := t.f.ReadAt(buf, int64(off)); err != nil {
				return fmt.Errorf("mp4: ses verisi okunamadi (frame %d): %w", idx, err)
			}
			if err := fn(buf); err != nil {
				return err
			}
			off += uint64(size)
			idx++
		}
	}
	return nil
}

// bitReader reads MSB-first bit fields, as the ASC bitstream requires.
type bitReader struct {
	data []byte
	pos  int
	err  bool
}

func (r *bitReader) read(n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		if r.pos >= len(r.data)*8 {
			r.err = true
			return v
		}
		bit := (r.data[r.pos/8] >> (7 - uint(r.pos%8))) & 1
		v = v<<1 | uint32(bit)
		r.pos++
	}
	return v
}

// IsMP4 reports whether path looks like an ISO base media file (MP4/M4A).
func IsMP4(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	var hdr [8]byte
	if _, err := io.ReadFull(f, hdr[:]); err != nil {
		return false
	}
	return string(hdr[4:8]) == "ftyp"
}

// Describe returns a short human-readable summary, used in error messages.
func (t *Track) Describe() string {
	return fmt.Sprintf("AAC-LC %d Hz %d kanal, %d frame", t.sampleRate, t.channels, t.frames)
}

// CodecName reports the codec of the audio track for logging: always AAC here,
// but the helper keeps call sites readable.
func CodecName() string { return strings.ToUpper("aac") }
