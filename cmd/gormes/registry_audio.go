//go:build !gormes_lite && !slim

package main

import (
	"path/filepath"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func registerAudioTools(reg *tools.Registry, cfg config.Config) {
	ttsProviders := map[string]tools.TTSProvider{}
	if edge := tools.NewEdgeTTSCommandProviderFromEnv(); edge != nil {
		ttsProviders["edge"] = edge
	}
	tools.RegisterTTSProviders(ttsProviders, tools.TTSProviderConfig{})
	reg.MustRegister(tools.NewTextToSpeechTool(tools.NewTTSRunner(tools.TTSConfig{
		OutputDir: filepath.Join(config.GormesHome(), "cache", "audio"),
		Provider:  configuredTTSProvider(cfg),
	}, ttsProviders)))
}

func audioToolsEnabled() bool { return true }
