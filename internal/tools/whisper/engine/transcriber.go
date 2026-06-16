package engine

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/engine/wasi"
)

const (
	TranscriberModelUnavailable = wasi.TranscriberModelUnavailable
	TranscriberWAVUnsupported   = wasi.TranscriberWAVUnsupported
	TranscriberWASIInference    = wasi.TranscriberWASIInference
	TranscriberClosed           = wasi.TranscriberClosed
)

type TranscriberError = wasi.TranscriberError
type Transcriber = wasi.Transcriber

func NewTranscriber(ctx context.Context, modelPath string, wasm []byte) (*Transcriber, error) {
	return wasi.NewTranscriber(ctx, modelPath, wasm)
}

func DecodePCM16Mono16kWAV(path string) ([]float32, error) {
	return wasi.DecodePCM16Mono16kWAV(path)
}
