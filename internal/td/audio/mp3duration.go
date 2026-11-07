package audio

import (
	"encoding/binary"
	"fmt"
	"os"
)

// MP3DurationSec returns playing time of an MP3 file in seconds.
func MP3DurationSec(path string) (float64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	off := 0
	if len(data) > 10 && data[0] == 'I' && data[1] == 'D' && data[2] == '3' {
		size := int(data[6]&0x7f)<<21 | int(data[7]&0x7f)<<14 | int(data[8]&0x7f)<<7 | int(data[9]&0x7f)
		off = 10 + size
		if off < 0 || off >= len(data) {
			return 0, fmt.Errorf("bad id3v2 size")
		}
	}

	frames := 0
	samples := 0
	sampleRate := 0
	pos := off
	for pos+4 <= len(data) {
		if data[pos] != 0xff || data[pos+1]&0xe0 != 0xe0 {
			pos++
			continue
		}
		h := binary.BigEndian.Uint32(data[pos : pos+4])
		ver := int((h >> 19) & 0x3)
		layer := int((h >> 17) & 0x3)
		brIdx := int((h >> 12) & 0xf)
		srIdx := int((h >> 10) & 0x3)
		pad := int((h >> 9) & 0x1)
		if ver == 1 || layer == 0 || brIdx == 0 || brIdx == 15 || srIdx == 3 {
			pos++
			continue
		}
		normLayer := 4 - layer
		brTab, ok := mp3Bitrates[ver][normLayer]
		if !ok {
			pos++
			continue
		}
		bitrate := brTab[brIdx] * 1000
		sr := mp3SampleRates[ver][srIdx]
		if sampleRate == 0 {
			sampleRate = sr
		}

		if frames == 0 {
			if n, ok := mp3XingFrames(data, pos, ver, normLayer); ok && n > 0 {
				sp := 1152
				if normLayer == 1 {
					sp = 384
				} else if normLayer == 3 && ver != 3 {
					sp = 576
				}
				return float64(n*sp) / float64(sr), nil
			}
		}

		var frameLen, samplesPerFrame int
		switch normLayer {
		case 1:
			frameLen = (12*bitrate/sr + pad) * 4
			samplesPerFrame = 384
		case 3:
			if ver == 3 {
				frameLen = 144*bitrate/sr + pad
				samplesPerFrame = 1152
			} else {
				frameLen = 72*bitrate/sr + pad
				samplesPerFrame = 576
			}
		default:
			frameLen = 144*bitrate/sr + pad
			samplesPerFrame = 1152
		}
		if frameLen <= 0 || pos+frameLen > len(data) {
			break
		}
		frames++
		samples += samplesPerFrame
		pos += frameLen
	}
	if frames == 0 || sampleRate == 0 {
		return 0, fmt.Errorf("no mp3 frames found")
	}
	return float64(samples) / float64(sampleRate), nil
}

func mp3XingFrames(data []byte, framePos, ver, layer int) (int, bool) {
	var xingOff int
	if ver == 3 {
		xingOff = 32 + 4
	} else {
		xingOff = 17 + 4
	}
	p := framePos + xingOff
	if p+12 > len(data) {
		return 0, false
	}
	tag := string(data[p : p+4])
	if tag != "Xing" && tag != "Info" {
		return 0, false
	}
	flags := binary.BigEndian.Uint32(data[p+4 : p+8])
	if flags&0x1 == 0 {
		return 0, false
	}
	n := int(binary.BigEndian.Uint32(data[p+8 : p+12]))
	return n, true
}

var mp3Bitrates = map[int]map[int][]int{
	3: {
		1: {0, 32, 64, 96, 128, 160, 192, 224, 256, 288, 320, 352, 384, 416, 448},
		2: {0, 32, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384},
		3: {0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320},
	},
	2: {
		1: {0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256},
		2: {0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},
		3: {0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},
	},
	0: {
		1: {0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256},
		2: {0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},
		3: {0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160},
	},
}

var mp3SampleRates = map[int][]int{
	3: {44100, 48000, 32000},
	2: {22050, 24000, 16000},
	0: {11025, 12000, 8000},
}
