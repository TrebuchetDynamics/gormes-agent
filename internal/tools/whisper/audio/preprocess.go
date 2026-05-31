package audio

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio/contract"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio/preprocessor"
)

const AudioPreprocessUnavailable = contract.AudioPreprocessUnavailable

type PCM = contract.PCM
type PreprocessError = contract.PreprocessError
type Converter = contract.Converter
type PreprocessOptions = contract.PreprocessOptions

func Preprocess(ctx context.Context, audioBytes []byte, mediaType string, opts PreprocessOptions) (PCM, error) {
	return preprocessor.Preprocess(ctx, audioBytes, mediaType, opts)
}
