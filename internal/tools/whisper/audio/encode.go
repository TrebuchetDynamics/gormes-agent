package audio

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio/codec"

func EncodePCM16MonoWAV(pcm PCM) ([]byte, error) {
	return codec.EncodePCM16MonoWAV(pcm)
}
