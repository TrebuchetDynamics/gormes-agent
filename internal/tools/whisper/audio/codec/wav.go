package codec

import (
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio/contract"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/wavpcm"
)

func DecodePCM16Mono16kWAV(raw []byte, label string) (contract.PCM, error) {
	pcm, err := wavpcm.DecodePCM16Mono16kWAV(raw)
	if err != nil {
		return contract.PCM{}, &contract.PreprocessError{Code: contract.AudioPreprocessUnavailable, Path: label, Err: err}
	}
	return pcm, nil
}

func EncodePCM16MonoWAV(pcm contract.PCM) ([]byte, error) {
	if pcm.SampleRate <= 0 {
		return nil, &contract.PreprocessError{Code: contract.AudioPreprocessUnavailable, Err: fmt.Errorf("sample rate is required")}
	}
	return wavpcm.EncodePCM16MonoWAV(pcm)
}
