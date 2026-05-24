package audio

import (
	"encoding/binary"
	"fmt"
)

func EncodePCM16MonoWAV(pcm PCM) ([]byte, error) {
	if pcm.SampleRate <= 0 {
		return nil, &PreprocessError{Code: AudioPreprocessUnavailable, Err: fmt.Errorf("sample rate is required")}
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
