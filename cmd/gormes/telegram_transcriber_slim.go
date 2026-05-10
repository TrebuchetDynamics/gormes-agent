//go:build slim

package main

import "github.com/TrebuchetDynamics/gormes-agent/internal/channels/telegram"

// resolveTelegramAudioTranscriber returns only the local-CLI transcriber in
// slim builds. The HTTP STT providers in internal/tools are excluded from
// slim binaries to keep the footprint minimal, so there is no fallback to
// wire here.
func resolveTelegramAudioTranscriber() telegram.AudioTranscriber {
	return telegram.NewWhisperTranscriberFromEnv()
}
