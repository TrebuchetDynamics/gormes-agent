//go:build !slim

package main

import (
	"github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/audiotools"
	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/telegram"
)

func resolveTelegramAudioTranscriber() telegram.AudioTranscriber {
	return audiotools.ResolveTelegramAudioTranscriber()
}
