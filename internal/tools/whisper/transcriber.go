package whisper

import (
	"context"
	_ "embed"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/engine"
)

const (
	TranscriberModelUnavailable = engine.TranscriberModelUnavailable
	TranscriberWAVUnsupported   = engine.TranscriberWAVUnsupported
	TranscriberWASIInference    = engine.TranscriberWASIInference
	TranscriberClosed           = engine.TranscriberClosed
)

//go:embed testdata/whisper.wasm
var whisperWASM []byte

type TranscriberError = engine.TranscriberError
type Transcriber = engine.Transcriber

func NewTranscriber(ctx context.Context, modelPath string) (*Transcriber, error) {
	return engine.NewTranscriber(ctx, modelPath, whisperWASM)
}

func decodePCM16Mono16kWAV(path string) ([]float32, error) {
	return engine.DecodePCM16Mono16kWAV(path)
}
