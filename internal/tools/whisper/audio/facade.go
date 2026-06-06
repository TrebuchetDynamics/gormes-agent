package audio

import (
	"context"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio/chunking"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio/codec"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio/contract"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio/converter"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio/normalization"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio/preprocessor"
)

const AudioPreprocessUnavailable = contract.AudioPreprocessUnavailable

type PCM = contract.PCM
type PreprocessError = contract.PreprocessError
type Converter = contract.Converter
type PreprocessOptions = contract.PreprocessOptions

// ChunkPCM splits PCM samples into chunks capped by maxChunk.
func ChunkPCM(samples []int16, sampleRate int, maxChunk time.Duration) [][]int16 {
	return chunking.ChunkPCM(samples, sampleRate, maxChunk)
}

// ChunkPCMWithOverlap splits PCM samples into chunks capped by maxChunk and overlaps adjacent chunks.
func ChunkPCMWithOverlap(samples []int16, sampleRate int, maxChunk, overlap time.Duration) [][]int16 {
	return chunking.ChunkPCMWithOverlap(samples, sampleRate, maxChunk, overlap)
}

// ConvertWithFFmpeg converts audio at inputPath to a PCM16 mono 16kHz WAV at outputPath.
func ConvertWithFFmpeg(ctx context.Context, inputPath, outputPath string) error {
	return converter.ConvertWithFFmpeg(ctx, inputPath, outputPath)
}

func EncodePCM16MonoWAV(pcm PCM) ([]byte, error) {
	return codec.EncodePCM16MonoWAV(pcm)
}

func NormalizeSpeechPCM(pcm PCM) PCM {
	return normalization.NormalizeSpeechPCM(pcm)
}

func Preprocess(ctx context.Context, audioBytes []byte, mediaType string, opts PreprocessOptions) (PCM, error) {
	return preprocessor.Preprocess(ctx, audioBytes, mediaType, opts)
}
