//go:build !slim

package tools

import (
	"context"
	"errors"
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/wasi/whisper/audio"
)

// AudioDecodeConfig controls audio decode behavior.
type AudioDecodeConfig struct{}

// DecodeOGGToPCM converts OGG/Opus bytes to PCM16 mono 16kHz samples
// using the build-tag-gated converter (ConvertWithFFmpeg or noffmpeg stub).
func DecodeOGGToPCM(ctx context.Context, oggData []byte, cfg AudioDecodeConfig) ([]int16, error) {
	if len(oggData) == 0 {
		return nil, fmt.Errorf("audio_decode_format_unsupported: empty audio input")
	}
	pcm, err := audio.Preprocess(ctx, oggData, "audio/ogg", audio.PreprocessOptions{
		FileName: "voice.ogg",
	})
	if err != nil {
		var pErr *audio.PreprocessError
		if errors.As(err, &pErr) {
			return nil, fmt.Errorf("audio_decode_ffmpeg_missing: %w", pErr.Unwrap())
		}
		return nil, fmt.Errorf("audio_decode_failed: %w", err)
	}
	return pcm.Samples, nil
}
