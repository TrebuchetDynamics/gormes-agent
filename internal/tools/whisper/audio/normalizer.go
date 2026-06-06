package audio

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio/normalization"

func NormalizeSpeechPCM(pcm PCM) PCM {
	return normalization.NormalizeSpeechPCM(pcm)
}
