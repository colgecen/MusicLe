package encode

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"MusicLeCLI/internal/td/audio"
)

// oggPackets extracts Opus packets from an Ogg Opus stream (test fixture only).
func oggPackets(data []byte) [][]byte {
	var packets [][]byte
	var pending []byte
	for pos := 0; pos+27 <= len(data); {
		if string(data[pos:pos+4]) != "OggS" {
			pos++
			continue
		}
		segments := int(data[pos+26])
		if pos+27+segments > len(data) {
			break
		}
		lacing := data[pos+27 : pos+27+segments]
		payload := pos + 27 + segments
		continued := data[pos+5]&0x01 != 0
		if !continued {
			pending = nil
		}
		for _, l := range lacing {
			end := payload + int(l)
			if end > len(data) {
				return packets
			}
			pending = append(pending, data[payload:end]...)
			payload = end
			if l < 255 {
				pkt := make([]byte, len(pending))
				copy(pkt, pending)
				packets = append(packets, pkt)
				pending = nil
			}
		}
		pos = payload
	}
	return packets
}

func TestEncodeOpusPacketsDurationMatches(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "testdata", "sample.opus"))
	if err != nil {
		t.Skipf("fixture yok: %v", err)
	}
	all := oggPackets(raw)
	var packets [][]byte
	for _, p := range all {
		if bytes.HasPrefix(p, []byte("OpusHead")) || bytes.HasPrefix(p, []byte("OpusTags")) {
			continue
		}
		packets = append(packets, p)
	}
	if len(packets) == 0 {
		t.Fatal("no opus packets parsed")
	}

	var out bytes.Buffer
	idx := 0
	next := func() ([]byte, error) {
		if idx >= len(packets) {
			return nil, io.EOF
		}
		p := packets[idx]
		idx++
		return p, nil
	}
	samples, err := encodeOpusPackets(context.Background(), next, &out, 0, nil)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if samples == 0 || out.Len() == 0 {
		t.Fatal("empty output")
	}

	dst := filepath.Join(t.TempDir(), "out.mp3")
	if err := os.WriteFile(dst, out.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	sec, err := audio.MP3DurationSec(dst)
	if err != nil {
		t.Fatalf("duration: %v", err)
	}
	want := float64(samples) / opusSampleRate
	// One padded MPEG frame at the end is allowed, nothing more.
	if diff := sec - want; diff < -0.05 || diff > 0.15 {
		t.Errorf("mp3 duration %.3fs, expected ~%.3fs", sec, want)
	}
}

func TestToMP3RejectsUnsupported(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "out.mp3")
	err := ToMP3(context.Background(), filepath.Join("..", "testdata", "sample.flac"), dst, 0, Tags{}, nil)
	if err == nil {
		t.Fatal("expected error for FLAC input")
	}
	if _, statErr := os.Stat(dst); statErr == nil {
		t.Error("partial output left behind")
	}
}

// TestToMP3FromM4A exercises the MP4/AAC path end to end on the fixture.
func TestToMP3FromM4A(t *testing.T) {
	src := filepath.Join("..", "testdata", "sample.m4a")
	if _, err := os.Stat(src); err != nil {
		t.Skipf("fixture yok: %v", err)
	}
	dst := filepath.Join(t.TempDir(), "out.mp3")
	if err := ToMP3(context.Background(), src, dst, 0, Tags{Title: "t"}, nil); err != nil {
		t.Fatalf("ToMP3: %v", err)
	}
	sec, err := audio.MP3DurationSec(dst)
	if err != nil {
		t.Fatalf("duration: %v", err)
	}
	// The fixture is a short clip; anything non-zero and sane passes.
	if sec <= 0 || sec > 600 {
		t.Errorf("unexpected duration %.2fs", sec)
	}
	if st, err := os.Stat(dst); err != nil || st.Size() == 0 {
		t.Error("empty output")
	}
}

func TestIsWebM(t *testing.T) {
	if IsWebM(filepath.Join("..", "testdata", "sample.opus")) {
		t.Error("ogg reported as webm")
	}
	if IsWebM(filepath.Join("..", "testdata", "does-not-exist")) {
		t.Error("missing file reported as webm")
	}
}
