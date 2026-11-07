package mp4audio

import (
	"path/filepath"
	"testing"
)

func TestParseASC(t *testing.T) {
	cases := []struct {
		name       string
		asc        []byte
		wantRate   int
		wantCh     int
		wantLC     bool
		wantHE     bool
		wantObjTyp int
	}{
		// 0x12 0x10: AOT 2 (AAC-LC), index 4 (44100), channel config 2.
		{"lc 44.1 stereo", []byte{0x12, 0x10}, 44100, 2, true, false, 2},
		// 0x11 0x90: AOT 2, index 3 (48000), channel config 1 (mono).
		{"lc 48k mono", []byte{0x11, 0x88}, 48000, 1, true, false, 2},
		// 0x13 0x10: AOT 2, index 6 (24000).
		{"lc 24k stereo", []byte{0x13, 0x10}, 24000, 2, true, false, 2},
		// 0x2B 0x92... AOT 5 is HE-AAC (SBR), not decodable here.
		{"he-aac", []byte{0x2B, 0x92, 0x08, 0x00}, 22050, 2, false, true, 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg, err := ParseASC(c.asc)
			if err != nil {
				t.Fatalf("ParseASC: %v", err)
			}
			if cfg.ObjectType != c.wantObjTyp {
				t.Errorf("ObjectType = %d, want %d", cfg.ObjectType, c.wantObjTyp)
			}
			if cfg.SampleRate != c.wantRate {
				t.Errorf("SampleRate = %d, want %d", cfg.SampleRate, c.wantRate)
			}
			if cfg.Channels != c.wantCh {
				t.Errorf("Channels = %d, want %d", cfg.Channels, c.wantCh)
			}
			if cfg.AACLC != c.wantLC {
				t.Errorf("AACLC = %v, want %v", cfg.AACLC, c.wantLC)
			}
			if cfg.HEAAC != c.wantHE {
				t.Errorf("HEAAC = %v, want %v", cfg.HEAAC, c.wantHE)
			}
		})
	}
}

func TestParseASCTooShort(t *testing.T) {
	if _, err := ParseASC([]byte{0x12}); err == nil {
		t.Error("expected an error for a 1-byte ASC")
	}
}

func TestFreqIndexFor(t *testing.T) {
	cases := map[int]int{96000: 0, 48000: 3, 44100: 4, 32000: 5, 8000: 11, 12345: 15}
	for rate, want := range cases {
		if got := FreqIndexFor(rate); got != want {
			t.Errorf("FreqIndexFor(%d) = %d, want %d", rate, got, want)
		}
	}
}

func TestADTSHeader(t *testing.T) {
	h := ADTSHeader(100, 4, 2) // 44.1 kHz stereo, payload 100
	if len(h) != 7 {
		t.Fatalf("header len = %d, want 7", len(h))
	}
	if h[0] != 0xFF || h[1] != 0xF1 {
		t.Errorf("sync word = %02x %02x, want ff f1", h[0], h[1])
	}
	// profile 1 (AAC-LC), freq index 4, channel config 2 -> 0x50
	if h[2] != 0x50 {
		t.Errorf("byte2 = %02x, want 50", h[2])
	}
	// frame length must be payload + header
	frameLen := int(h[3]&0x03)<<11 | int(h[4])<<3 | int(h[5])>>5
	if frameLen != 107 {
		t.Errorf("frame length = %d, want 107", frameLen)
	}
	// channel config low bits live in byte 3's top two bits
	if h[3]>>6 != 2 {
		t.Errorf("channels in byte3 = %d, want 2", h[3]>>6)
	}
}

func TestOpenFixture(t *testing.T) {
	path := filepath.Join("..", "testdata", "sample.m4a")
	tr, err := Open(path)
	if err != nil {
		t.Skipf("fixture yok: %v", err)
	}
	defer tr.Close()

	if tr.SampleRate() <= 0 || tr.Channels() <= 0 || tr.Frames() == 0 {
		t.Fatalf("bad track info: %s", tr.Describe())
	}
	if tr.DurationSec() <= 0 {
		t.Errorf("DurationSec = %v", tr.DurationSec())
	}

	frames := 0
	err = tr.ForEachFrame(func(au []byte) error {
		if len(au) == 0 {
			t.Fatal("empty access unit")
		}
		frames++
		return nil
	})
	if err != nil {
		t.Fatalf("ForEachFrame: %v", err)
	}
	if frames != tr.Frames() {
		t.Errorf("iterated %d frames, track reports %d", frames, tr.Frames())
	}
}

func TestOpenRejectsNonMP4(t *testing.T) {
	if _, err := Open(filepath.Join("..", "testdata", "sample.mp3")); err == nil {
		t.Error("expected an error for an MP3")
	}
}

func TestIsMP4(t *testing.T) {
	if !IsMP4(filepath.Join("..", "testdata", "sample.m4a")) {
		t.Error("m4a fixture not recognised")
	}
	if IsMP4(filepath.Join("..", "testdata", "sample.mp3")) {
		t.Error("mp3 reported as mp4")
	}
	if IsMP4(filepath.Join("..", "testdata", "nope.mp4")) {
		t.Error("missing file reported as mp4")
	}
}
