package audio

import (
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/wavpcm"
)

func EncodePCM16MonoWAV(pcm PCM) ([]byte, error) {
	if pcm.SampleRate <= 0 {
		return nil, &PreprocessError{Code: AudioPreprocessUnavailable, Err: fmt.Errorf("sample rate is required")}
	}
	return wavpcm.EncodePCM16MonoWAV(pcm)
}
