//go:build !noffmpeg

package audio

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/pathredact"
)

// ConvertWithFFmpeg converts audio at inputPath to a PCM16 mono 16kHz WAV at outputPath.
// It requires ffmpeg on the system PATH and is excluded from noffmpeg builds.
func ConvertWithFFmpeg(ctx context.Context, inputPath, outputPath string) error {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return &PreprocessError{Code: AudioPreprocessUnavailable, Path: filepath.Base(inputPath), Err: errors.New("ffmpeg not found")}
	}
	cmd := exec.CommandContext(ctx, ffmpeg, "-y", "-i", inputPath, "-ar", "16000", "-ac", "1", outputPath)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return &PreprocessError{Code: AudioPreprocessUnavailable, Path: filepath.Base(inputPath), Err: ctx.Err()}
	}
	detail := strings.TrimSpace(string(out))
	if detail == "" {
		detail = err.Error()
	}
	if len(detail) > 300 {
		detail = detail[:300] + "...(truncated)"
	}
	detail = pathredact.Text(detail, inputPath, outputPath)
	return &PreprocessError{Code: AudioPreprocessUnavailable, Path: filepath.Base(inputPath), Err: fmt.Errorf("ffmpeg failed: %s", detail)}
}
