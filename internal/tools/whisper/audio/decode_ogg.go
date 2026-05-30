//go:build !slim

package audio

import (
	"context"
	"errors"
	"fmt"
)

// DecodeConfig controls audio decode behavior.
type DecodeConfig struct{}

// DecodeOGGToPCM converts OGG/Opus bytes to PCM16 mono 16kHz samples
// using the build-tag-gated converter (ConvertWithFFmpeg or noffmpeg stub).
func DecodeOGGToPCM(ctx context.Context, oggData []byte, cfg DecodeConfig) ([]int16, error) {
	if len(oggData) == 0 {
		return nil, fmt.Errorf("audio_decode_format_unsupported: empty audio input")
	}
	pcm, err := Preprocess(ctx, oggData, "audio/ogg", PreprocessOptions{
		FileName: "voice.ogg",
	})
	if err != nil {
		var pErr *PreprocessError
		if errors.As(err, &pErr) {
			return nil, fmt.Errorf("audio_decode_ffmpeg_missing: %w", pErr.Unwrap())
		}
		return nil, fmt.Errorf("audio_decode_failed: %w", err)
	}
	return pcm.Samples, nil
}
