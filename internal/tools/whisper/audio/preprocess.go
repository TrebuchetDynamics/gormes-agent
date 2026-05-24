package audio

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const AudioPreprocessUnavailable = "audio_preprocess_unavailable"

type PCM struct {
	Samples    []int16
	SampleRate int
}

type PreprocessError struct {
	Code string
	Path string
	Err  error
}

func (e *PreprocessError) Error() string {
	var parts []string
	parts = append(parts, e.Code)
	if e.Path != "" {
		parts = append(parts, "path="+filepath.Base(e.Path))
	}
	if e.Err != nil {
		parts = append(parts, e.Err.Error())
	}
	return strings.Join(parts, ": ")
}

func (e *PreprocessError) Unwrap() error {
	return e.Err
}

type Converter func(context.Context, string, string) error

type PreprocessOptions struct {
	FileName  string
	Converter Converter
}

func Preprocess(ctx context.Context, audioBytes []byte, mediaType string, opts PreprocessOptions) (PCM, error) {
	if len(audioBytes) == 0 {
		return PCM{}, &PreprocessError{Code: AudioPreprocessUnavailable, Err: errors.New("audio input is empty")}
	}

	ext := audioExtension(mediaType, opts.FileName)
	if isWAVExtension(ext) {
		return decodePCM16Mono16kWAV(audioBytes, filepath.Base(firstNonEmpty(opts.FileName, "input"+ext)))
	}

	dir, err := os.MkdirTemp("", "gormes-wasi-whisper-audio-*")
	if err != nil {
		return PCM{}, &PreprocessError{Code: AudioPreprocessUnavailable, Err: fmt.Errorf("create tempdir: %w", err)}
	}
	defer os.RemoveAll(dir)

	inputPath := filepath.Join(dir, "input"+ext)
	outputPath := filepath.Join(dir, "input.wav")
	if err := os.WriteFile(inputPath, audioBytes, 0o600); err != nil {
		return PCM{}, &PreprocessError{Code: AudioPreprocessUnavailable, Path: filepath.Base(inputPath), Err: fmt.Errorf("write input: %w", err)}
	}

	converter := opts.Converter
	if converter == nil {
		converter = ConvertWithFFmpeg
	}
	if err := converter(ctx, inputPath, outputPath); err != nil {
		var preprocessErr *PreprocessError
		if errors.As(err, &preprocessErr) {
			return PCM{}, preprocessErr
		}
		return PCM{}, &PreprocessError{Code: AudioPreprocessUnavailable, Path: filepath.Base(inputPath), Err: redactPathError(err, inputPath, outputPath)}
	}
	raw, err := os.ReadFile(outputPath)
	if err != nil {
		return PCM{}, &PreprocessError{Code: AudioPreprocessUnavailable, Path: filepath.Base(outputPath), Err: fmt.Errorf("read converted wav: %w", err)}
	}
	return decodePCM16Mono16kWAV(raw, filepath.Base(outputPath))
}

func decodePCM16Mono16kWAV(raw []byte, label string) (PCM, error) {
	if len(raw) < 12 || string(raw[0:4]) != "RIFF" || string(raw[8:12]) != "WAVE" {
		return PCM{}, &PreprocessError{Code: AudioPreprocessUnavailable, Path: label, Err: errors.New("not a RIFF/WAVE file")}
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
			return PCM{}, &PreprocessError{Code: AudioPreprocessUnavailable, Path: label, Err: fmt.Errorf("truncated %s chunk", chunkID)}
		}
		switch chunkID {
		case "fmt ":
			if chunkSize < 16 {
				return PCM{}, &PreprocessError{Code: AudioPreprocessUnavailable, Path: label, Err: errors.New("short fmt chunk")}
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
		return PCM{}, &PreprocessError{Code: AudioPreprocessUnavailable, Path: label, Err: errors.New("missing fmt or data chunk")}
	}
	if audioFormat != 1 || numChannels != 1 || sampleRate != 16000 || bitsPerSample != 16 {
		return PCM{}, &PreprocessError{Code: AudioPreprocessUnavailable, Path: label, Err: fmt.Errorf("want PCM16 mono 16000Hz, got format=%d channels=%d sample_rate=%d bits=%d", audioFormat, numChannels, sampleRate, bitsPerSample)}
	}
	if len(data)%2 != 0 {
		return PCM{}, &PreprocessError{Code: AudioPreprocessUnavailable, Path: label, Err: errors.New("odd PCM byte length")}
	}

	samples := make([]int16, len(data)/2)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(data[i*2 : i*2+2]))
	}
	return PCM{Samples: samples, SampleRate: int(sampleRate)}, nil
}

func audioExtension(mediaType, fileName string) string {
	if ext := strings.ToLower(strings.TrimSpace(filepath.Ext(fileName))); ext != "" && len(ext) <= 10 {
		return ext
	}
	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "audio/wav", "audio/wave", "audio/x-wav":
		return ".wav"
	case "audio/ogg", "audio/opus":
		return ".ogg"
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	default:
		return ".ogg"
	}
}

func isWAVExtension(ext string) bool {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ".wav", ".wave":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func redactPathError(err error, paths ...string) error {
	if err == nil {
		return nil
	}
	return errors.New(redactPathText(err.Error(), paths...))
}

func redactPathText(text string, paths ...string) string {
	redacted := text
	for _, path := range paths {
		if path == "" {
			continue
		}
		redacted = strings.ReplaceAll(redacted, path, filepath.Base(path))
	}
	return redacted
}
