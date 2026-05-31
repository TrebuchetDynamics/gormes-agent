//go:build !slim

package audio

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio/decoder"
)

// DecodeConfig controls audio decode behavior.
type DecodeConfig = decoder.DecodeConfig

// DecodeOGGToPCM converts OGG/Opus bytes to PCM16 mono 16kHz samples
// using the build-tag-gated converter (ConvertWithFFmpeg or noffmpeg stub).
func DecodeOGGToPCM(ctx context.Context, oggData []byte, cfg DecodeConfig) ([]int16, error) {
	return decoder.DecodeOGGToPCM(ctx, oggData, cfg)
}
