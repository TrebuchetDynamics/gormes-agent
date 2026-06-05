//go:build !slim

package tools

import (
	"context"

	whisperaudio "github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio"
)

// AudioDecodeConfig controls audio decode behavior.
type AudioDecodeConfig = whisperaudio.DecodeConfig

// DecodeOGGToPCM converts OGG/Opus bytes to PCM16 mono 16kHz samples
// using the build-tag-gated converter (ConvertWithFFmpeg or noffmpeg stub).
func DecodeOGGToPCM(ctx context.Context, oggData []byte, cfg AudioDecodeConfig) ([]int16, error) {
	return whisperaudio.DecodeOGGToPCM(ctx, oggData, cfg)
}
