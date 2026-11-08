package encode

import (
	"context"
	"fmt"
	"os"

	"github.com/tphakala/go-aac/pcm"

	"MusicLe/internal/td/mp4audio"
)

// aacToMP3 is the MP4/AAC half of ToMP3. It extracts every AAC access unit from
// the container, wraps it in an ADTS header and decodes it to PCM with the pure
// Go go-aac decoder, then re-encodes that PCM as MP3.
//
// YouTube only offers audio-only Opus streams to clients that carry a PO token,
// while the combined A/V stream (itag 18) still downloads and carries AAC-LC.
// That makes this path the fallback that keeps most videos downloadable.
func aacToMP3(ctx context.Context, src, dst string, tags Tags, progress func(float64)) error {
	track, err := mp4audio.Open(src)
	if err != nil {
		return err
	}
	defer track.Close()

	if track.SampleRate() <= 0 || track.Channels() <= 0 {
		return ErrUnsupportedSource
	}
	// The MP3 encoder needs a supported MPEG-1 rate; the AAC rates we care
	// about (32/44.1/48 kHz) already are.
	if !mp3SupportsRate(track.SampleRate()) {
		return fmt.Errorf("desteklenmeyen ornekleme hizi: %d Hz", track.SampleRate())
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	success := false
	defer func() {
		_ = out.Close()
		if !success {
			_ = os.Remove(dst)
		}
	}()

	channels := track.Channels()
	if channels < 1 || channels > mp3Channels {
		return fmt.Errorf("desteklenmeyen kanal sayisi: %d", channels)
	}
	w := newMP3Writer(out, track.SampleRate(), channels, track.DurationSec(), progress)

	dec := pcm.NewADTSDecoder()
	freqIndex := mp4audio.FreqIndexFor(track.SampleRate())
	adts := make([]byte, 0, 8192)
	var pcmBuf []byte

	frames := 0
	var firstDecodeErr error
	err = track.ForEachFrame(func(au []byte) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		needed := len(au) + 7
		if cap(adts) < needed {
			adts = make([]byte, needed)
		}
		adts = adts[:needed]
		copy(adts, mp4audio.ADTSHeader(len(au), freqIndex, track.Channels()))
		copy(adts[7:], au)

		var derr error
		pcmBuf, _, derr = dec.DecodeFrame(pcmBuf[:0], adts)
		if derr != nil {
			// One corrupt frame must not kill the whole download, but if
			// nothing decodes at all the reason is useful in the error.
			if firstDecodeErr == nil {
				firstDecodeErr = derr
			}
			return nil
		}
		if len(pcmBuf) == 0 {
			return nil
		}
		if err := w.write(pcmToInt16(pcmBuf, channels)); err != nil {
			return err
		}
		frames++
		return nil
	})
	if err != nil {
		return err
	}
	if frames == 0 {
		if firstDecodeErr != nil {
			return fmt.Errorf("ses cozulemedi: %w", firstDecodeErr)
		}
		return fmt.Errorf("ses cozulemedi: AAC frame bulunamadi")
	}
	if err := w.close(); err != nil {
		return err
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

// pcmToInt16 converts the decoder's little-endian S16 output to int16 samples.
// When the source is mono a duplicate channel is inserted so the encoder, which
// always writes stereo, gets the layout it expects.
func pcmToInt16(pcm []byte, channels int) []int16 {
	n := len(pcm) / 2
	out := make([]int16, 0, n*2)
	for i := 0; i+1 < len(pcm); i += 2 {
		v := int16(uint16(pcm[i]) | uint16(pcm[i+1])<<8)
		out = append(out, v)
		if channels == 1 {
			out = append(out, v)
		}
	}
	return out
}

// mp3SupportsRate reports whether shine-mp3 can encode at the given rate.
func mp3SupportsRate(rate int) bool {
	switch rate {
	case 32000, 44100, 48000:
		return true
	}
	return false
}
