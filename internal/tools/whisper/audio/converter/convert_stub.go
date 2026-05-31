//go:build noffmpeg

package converter

import (
	"context"
	"errors"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio/contract"
)

// ConvertWithFFmpeg is a stub for noffmpeg builds. It returns a typed error
// indicating that ffmpeg is not available, rather than spawning a subprocess.
func ConvertWithFFmpeg(ctx context.Context, inputPath, outputPath string) error {
	return &contract.PreprocessError{
		Code: contract.AudioPreprocessUnavailable,
		Path: inputPath,
		Err:  errors.New("ffmpeg not available in noffmpeg build; install ffmpeg or rebuild without -tags noffmpeg"),
	}
}
