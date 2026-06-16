package preprocessor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio/codec"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio/contract"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio/converter"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio/format"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/pathredact"
)

func Preprocess(ctx context.Context, audioBytes []byte, mediaType string, opts contract.PreprocessOptions) (contract.PCM, error) {
	if len(audioBytes) == 0 {
		return contract.PCM{}, contract.NewUnavailableError("", errors.New("audio input is empty"))
	}

	ext := format.Extension(mediaType, opts.FileName)
	if format.IsWAVExtension(ext) {
		return codec.DecodePCM16Mono16kWAV(audioBytes, filepath.Base(firstNonEmpty(opts.FileName, "input"+ext)))
	}

	dir, err := os.MkdirTemp("", "gormes-wasi-whisper-audio-*")
	if err != nil {
		return contract.PCM{}, contract.NewUnavailableError("", fmt.Errorf("create tempdir: %w", err))
	}
	defer os.RemoveAll(dir)

	inputPath := filepath.Join(dir, "input"+ext)
	outputPath := filepath.Join(dir, "input.wav")
	if err := os.WriteFile(inputPath, audioBytes, 0o600); err != nil {
		return contract.PCM{}, contract.NewUnavailableError(filepath.Base(inputPath), fmt.Errorf("write input: %w", err))
	}

	convert := opts.Converter
	if convert == nil {
		convert = converter.ConvertWithFFmpeg
	}
	if err := convert(ctx, inputPath, outputPath); err != nil {
		var preprocessErr *contract.PreprocessError
		if errors.As(err, &preprocessErr) {
			return contract.PCM{}, preprocessErr
		}
		return contract.PCM{}, contract.NewUnavailableError(filepath.Base(inputPath), pathredact.Error(err, inputPath, outputPath))
	}
	raw, err := os.ReadFile(outputPath)
	if err != nil {
		return contract.PCM{}, contract.NewUnavailableError(filepath.Base(outputPath), fmt.Errorf("read converted wav: %w", err))
	}
	return codec.DecodePCM16Mono16kWAV(raw, filepath.Base(outputPath))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
