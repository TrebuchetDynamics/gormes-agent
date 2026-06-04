//go:build !gormes_lite && !slim

package main

import (
	"github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/audiotools"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func registerAudioTools(reg *tools.Registry, cfg config.Config) {
	audiotools.Register(reg, cfg, audiotools.Options{
		AudioCacheDir:         defaultAudioCacheDir(),
		TranscriptionCacheDir: defaultTranscriptionCacheDir(),
		RuntimeTTSProvider:    configuredTTSProvider(cfg),
	})
}

func configuredTranscriptionConfig(cfg config.Config) tools.TranscriptionConfig {
	return audiotools.ConfiguredTranscriptionConfig(cfg)
}

func audioToolsEnabled() bool { return true }

func configuredTTSProviderFromMap(ttsConfig map[string]any) string {
	return audiotools.ConfiguredTTSProviderFromMap(ttsConfig)
}
