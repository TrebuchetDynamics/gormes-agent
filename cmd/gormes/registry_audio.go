//go:build !gormes_lite && !slim

package main

import (
	"os"
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
		OutputDir:      defaultAudioCacheDir(),
		Provider:       provider,
		ProviderConfig: cfg.TTS,
	}, ttsProviders)))

	// Register STT transcription tool with local whisper as default.
	transcriptionCacheDir := defaultTranscriptionCacheDir()
	sttProviders := map[string]tools.TranscriptionProvider{}
	sttProviders["local"] = tools.NewLocalSTTProvider(transcriptionCacheDir)
	tools.RegisterTranscriptionProviders(sttProviders, tools.TranscriptionProviderConfig{
		APIKey:  os.Getenv("GORMES_STT_OPENAI_KEY"),
		BaseURL: os.Getenv("GORMES_STT_OPENAI_BASE_URL"),
		Model:   "",
	})
	reg.MustRegister(tools.NewTranscriptionTool(tools.NewTranscriptionRunner(configuredTranscriptionConfig(cfg), sttProviders)))
}

func configuredTranscriptionConfig(cfg config.Config) tools.TranscriptionConfig {
	return tools.TranscriptionConfig{
		Disabled:    false,
		Provider:    strings.ToLower(strings.TrimSpace(cfg.STT.Provider)),
		LocalModel:  strings.TrimSpace(cfg.STT.Local.Model),
		OpenAIModel: strings.TrimSpace(cfg.STT.OpenAI.Model),
		Language:    strings.TrimSpace(cfg.STT.Local.Language),
	}
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
