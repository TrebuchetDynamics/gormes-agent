//go:build !gormes_lite && !slim

package main

import (
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func registerAudioTools(reg *tools.Registry, cfg config.Config) {
	ttsProviders := map[string]tools.TTSProvider{}
	tools.RegisterTTSCommandProviders(ttsProviders, cfg.TTS, nil)
	if edge := tools.NewEdgeTTSCommandProviderFromEnv(); edge != nil {
		ttsProviders["edge"] = edge
	}
	tools.RegisterTTSProviders(ttsProviders, tools.TTSProviderConfig{ProviderConfig: cfg.TTS})
	provider := configuredTTSProvider(cfg)
	if configured := configuredTTSProviderFromMap(cfg.TTS); configured != "" {
		provider = configured
	}
	reg.MustRegister(tools.NewTextToSpeechTool(tools.NewTTSRunner(tools.TTSConfig{
		OutputDir:      filepath.Join(config.GormesHome(), "cache", "audio"),
		Provider:       provider,
		ProviderConfig: cfg.TTS,
	}, ttsProviders)))
}

func audioToolsEnabled() bool { return true }

func configuredTTSProviderFromMap(ttsConfig map[string]any) string {
	if ttsConfig == nil {
		return ""
	}
	value, ok := ttsConfig["provider"].(string)
	if !ok {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(value))
}
