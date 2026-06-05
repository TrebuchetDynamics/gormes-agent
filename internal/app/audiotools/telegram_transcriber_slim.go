//go:build slim

package audiotools

import "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/telegram"

// ResolveTelegramAudioTranscriber returns only the local-CLI transcriber in
// slim builds. The HTTP STT providers in internal/tools are excluded from
// slim binaries to keep the footprint minimal, so there is no fallback to
// wire here.
func ResolveTelegramAudioTranscriber() telegram.AudioTranscriber {
	return telegram.NewWhisperTranscriberFromEnv()
}
