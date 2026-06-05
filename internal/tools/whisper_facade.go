package tools

import (
	"context"
	"time"

	wasiwhisper "github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper"
	whisperaudio "github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio"
)

type WASIWhisperTranscriber interface {
	TranscribeWAV(ctx context.Context, path string) (string, error)
	Close(ctx context.Context) error
}

type WhisperAudioConverter = whisperaudio.Converter
type WhisperPCM = whisperaudio.PCM
type WhisperPreprocessOptions = whisperaudio.PreprocessOptions

func NewWASIWhisperTranscriber(ctx context.Context, modelPath string) (WASIWhisperTranscriber, error) {
	return wasiwhisper.NewTranscriber(ctx, modelPath)
}

func PreprocessWhisperAudio(ctx context.Context, data []byte, mediaType string, opts WhisperPreprocessOptions) (WhisperPCM, error) {
	return whisperaudio.Preprocess(ctx, data, mediaType, opts)
}

func ChunkWhisperPCM(samples []int16, sampleRate int, maxDuration time.Duration) [][]int16 {
	return whisperaudio.ChunkPCM(samples, sampleRate, maxDuration)
}

func EncodeWhisperPCM16MonoWAV(pcm WhisperPCM) ([]byte, error) {
	return whisperaudio.EncodePCM16MonoWAV(pcm)
}
