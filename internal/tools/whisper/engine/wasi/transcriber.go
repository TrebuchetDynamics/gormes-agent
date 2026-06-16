package wasi

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/engine/wasi/contract"
	wasiruntime "github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/engine/wasi/runtime"
)

const (
	TranscriberModelUnavailable = contract.TranscriberModelUnavailable
	TranscriberWAVUnsupported   = contract.TranscriberWAVUnsupported
	TranscriberWASIInference    = contract.TranscriberWASIInference
	TranscriberClosed           = contract.TranscriberClosed
)

type TranscriberError = contract.TranscriberError

type Transcriber = wasiruntime.Transcriber

func NewTranscriber(ctx context.Context, modelPath string, wasm []byte) (*Transcriber, error) {
	return wasiruntime.NewTranscriber(ctx, modelPath, wasm)
}

func DecodePCM16Mono16kWAV(path string) ([]float32, error) {
	return wasiruntime.DecodePCM16Mono16kWAV(path)
}
