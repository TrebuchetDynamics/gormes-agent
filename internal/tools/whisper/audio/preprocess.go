package audio

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio/contract"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/pathredact"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/wavpcm"
)

const AudioPreprocessUnavailable = contract.AudioPreprocessUnavailable

type PCM = contract.PCM
type PreprocessError = contract.PreprocessError
type Converter = contract.Converter
type PreprocessOptions = contract.PreprocessOptions

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
		return PCM{}, &PreprocessError{Code: AudioPreprocessUnavailable, Path: filepath.Base(inputPath), Err: pathredact.Error(err, inputPath, outputPath)}
	}
	raw, err := os.ReadFile(outputPath)
	if err != nil {
		return PCM{}, &PreprocessError{Code: AudioPreprocessUnavailable, Path: filepath.Base(outputPath), Err: fmt.Errorf("read converted wav: %w", err)}
	}
	return decodePCM16Mono16kWAV(raw, filepath.Base(outputPath))
}

func decodePCM16Mono16kWAV(raw []byte, label string) (PCM, error) {
	pcm, err := wavpcm.DecodePCM16Mono16kWAV(raw)
	if err != nil {
		return PCM{}, &PreprocessError{Code: AudioPreprocessUnavailable, Path: label, Err: err}
	}
	return pcm, nil
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
