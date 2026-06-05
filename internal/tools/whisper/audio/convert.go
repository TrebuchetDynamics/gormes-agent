package audio

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio/converter"
)

// ConvertWithFFmpeg converts audio at inputPath to a PCM16 mono 16kHz WAV at outputPath.
func ConvertWithFFmpeg(ctx context.Context, inputPath, outputPath string) error {
	return converter.ConvertWithFFmpeg(ctx, inputPath, outputPath)
}
