package contract

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/wavpcm"
)

const AudioPreprocessUnavailable = "audio_preprocess_unavailable"

type PCM = wavpcm.PCM

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
