//go:build !slim

package decoder

import (
	"context"
	"errors"
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio/contract"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio/preprocessor"
)

// DecodeConfig controls audio decode behavior.
type DecodeConfig struct{}

// DecodeOGGToPCM converts OGG/Opus bytes to PCM16 mono 16kHz samples
// using the build-tag-gated converter.
func DecodeOGGToPCM(ctx context.Context, oggData []byte, cfg DecodeConfig) ([]int16, error) {
	if len(oggData) == 0 {
		return nil, fmt.Errorf("audio_decode_format_unsupported: empty audio input")
	}
	pcm, err := preprocessor.Preprocess(ctx, oggData, "audio/ogg", contract.PreprocessOptions{
		FileName: "voice.ogg",
	})
	if err != nil {
		var pErr *contract.PreprocessError
		if errors.As(err, &pErr) {
			return nil, fmt.Errorf("audio_decode_ffmpeg_missing: %w", pErr.Unwrap())
		}
		return nil, fmt.Errorf("audio_decode_failed: %w", err)
	}
	return pcm.Samples, nil
}
