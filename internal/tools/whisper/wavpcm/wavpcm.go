package wavpcm

import (
	"encoding/binary"
	"errors"
	"fmt"
)

type PCM struct {
	Samples    []int16
	SampleRate int
}

func EncodePCM16MonoWAV(pcm PCM) ([]byte, error) {
	if pcm.SampleRate <= 0 {
		return nil, fmt.Errorf("sample rate is required")
	}
	dataSize := len(pcm.Samples) * 2
	raw := make([]byte, 44+dataSize)
	copy(raw[0:4], "RIFF")
	binary.LittleEndian.PutUint32(raw[4:8], uint32(36+dataSize))
	copy(raw[8:12], "WAVE")
	copy(raw[12:16], "fmt ")
	binary.LittleEndian.PutUint32(raw[16:20], 16)
	binary.LittleEndian.PutUint16(raw[20:22], 1)
	binary.LittleEndian.PutUint16(raw[22:24], 1)
	binary.LittleEndian.PutUint32(raw[24:28], uint32(pcm.SampleRate))
	binary.LittleEndian.PutUint32(raw[28:32], uint32(pcm.SampleRate*2))
	binary.LittleEndian.PutUint16(raw[32:34], 2)
	binary.LittleEndian.PutUint16(raw[34:36], 16)
	copy(raw[36:40], "data")
	binary.LittleEndian.PutUint32(raw[40:44], uint32(dataSize))
	for i, sample := range pcm.Samples {
		binary.LittleEndian.PutUint16(raw[44+(i*2):46+(i*2)], uint16(sample))
	}
	return raw, nil
}

func DecodePCM16Mono16kWAV(raw []byte) (PCM, error) {
	if len(raw) < 12 || string(raw[0:4]) != "RIFF" || string(raw[8:12]) != "WAVE" {
		return PCM{}, errors.New("not a RIFF/WAVE file")
	}

	var (
		haveFormat    bool
		audioFormat   uint16
		numChannels   uint16
		sampleRate    uint32
		bitsPerSample uint16
		data          []byte
	)
	for offset := 12; offset+8 <= len(raw); {
		chunkID := string(raw[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(raw[offset+4 : offset+8]))
		chunkStart := offset + 8
		chunkEnd := chunkStart + chunkSize
		if chunkSize < 0 || chunkEnd > len(raw) {
			return PCM{}, fmt.Errorf("truncated %s chunk", chunkID)
		}
		switch chunkID {
		case "fmt ":
			if chunkSize < 16 {
				return PCM{}, errors.New("short fmt chunk")
			}
			haveFormat = true
			audioFormat = binary.LittleEndian.Uint16(raw[chunkStart : chunkStart+2])
			numChannels = binary.LittleEndian.Uint16(raw[chunkStart+2 : chunkStart+4])
			sampleRate = binary.LittleEndian.Uint32(raw[chunkStart+4 : chunkStart+8])
			bitsPerSample = binary.LittleEndian.Uint16(raw[chunkStart+14 : chunkStart+16])
		case "data":
			data = raw[chunkStart:chunkEnd]
		}
		offset = chunkEnd
		if offset%2 == 1 {
			offset++
		}
	}
	if !haveFormat || len(data) == 0 {
		return PCM{}, errors.New("missing fmt or data chunk")
	}
	if audioFormat != 1 || numChannels != 1 || sampleRate != 16000 || bitsPerSample != 16 {
		return PCM{}, fmt.Errorf("want PCM16 mono 16000Hz, got format=%d channels=%d sample_rate=%d bits=%d", audioFormat, numChannels, sampleRate, bitsPerSample)
	}
	if len(data)%2 != 0 {
		return PCM{}, errors.New("odd PCM byte length")
	}

	samples := make([]int16, len(data)/2)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(data[i*2 : i*2+2]))
	}
	return PCM{Samples: samples, SampleRate: int(sampleRate)}, nil
}
