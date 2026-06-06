package wav

import (
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/engine/wasi/contract"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/wavpcm"
)

func DecodePCM16Mono16k(path string) ([]float32, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, &contract.TranscriberError{Code: contract.TranscriberWAVUnsupported, Path: filepath.Base(path), Err: err}
	}
	pcm, err := wavpcm.DecodePCM16Mono16kWAV(raw)
	if err != nil {
		return nil, &contract.TranscriberError{Code: contract.TranscriberWAVUnsupported, Path: filepath.Base(path), Err: err}
	}
	samples := make([]float32, len(pcm.Samples))
	for i, sample := range pcm.Samples {
		samples[i] = float32(sample) / 32768.0
	}
	return samples, nil
}
