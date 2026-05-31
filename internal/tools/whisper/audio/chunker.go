package audio

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio/chunking"
)

func ChunkPCM(samples []int16, sampleRate int, maxChunk time.Duration) [][]int16 {
	return chunking.ChunkPCM(samples, sampleRate, maxChunk)
}
