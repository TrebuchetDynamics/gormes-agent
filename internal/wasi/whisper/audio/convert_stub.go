//go:build noffmpeg

package audio

import (
	"context"
	"errors"
)

// ConvertWithFFmpeg is a stub for noffmpeg builds. It returns a typed error
// indicating that ffmpeg is not available, rather than spawning a subprocess.
func ConvertWithFFmpeg(ctx context.Context, inputPath, outputPath string) error {
	return &PreprocessError{
		Code: AudioPreprocessUnavailable,
		Path: inputPath,
		Err:  errors.New("ffmpeg not available in noffmpeg build; install ffmpeg or rebuild without -tags noffmpeg"),
	}
}
