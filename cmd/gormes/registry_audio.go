//go:build !gormes_lite && !slim

package main

import (
	"os"
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

	// Register STT transcription tool with local whisper as default.
	transcriptionCacheDir := filepath.Join(config.GormesHome(), "cache", "whisper")
	if override := os.Getenv("GORMES_STT_CACHE_DIR"); override != "" {
		transcriptionCacheDir = override
	}
	sttProviders := map[string]tools.TranscriptionProvider{}
	sttProviders["local"] = tools.NewLocalSTTProvider(transcriptionCacheDir)
	tools.RegisterTranscriptionProviders(sttProviders, tools.TranscriptionProviderConfig{
		APIKey:  os.Getenv("GORMES_STT_OPENAI_KEY"),
		BaseURL: os.Getenv("GORMES_STT_OPENAI_BASE_URL"),
		Model:   "",
	})
	reg.MustRegister(tools.NewTranscriptionTool(tools.NewTranscriptionRunner(tools.TranscriptionConfig{
		Disabled: false,
	}, sttProviders)))
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
