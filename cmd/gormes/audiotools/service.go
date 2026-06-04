//go:build !gormes_lite && !slim

package audiotools

import (
	"os"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type Options struct {
	AudioCacheDir         string
	TranscriptionCacheDir string
	RuntimeTTSProvider    string
}

func Register(reg *tools.Registry, cfg config.Config, opts Options) {
	ttsProviders := map[string]tools.TTSProvider{}
	tools.RegisterTTSCommandProviders(ttsProviders, cfg.TTS, nil)
	if edge := tools.NewEdgeTTSCommandProviderFromEnv(); edge != nil {
		ttsProviders["edge"] = edge
	}
	tools.RegisterTTSProviders(ttsProviders, tools.TTSProviderConfig{ProviderConfig: cfg.TTS})
	provider := opts.RuntimeTTSProvider
	if configured := ConfiguredTTSProviderFromMap(cfg.TTS); configured != "" {
		provider = configured
	}
	reg.MustRegister(tools.NewTextToSpeechTool(tools.NewTTSRunner(tools.TTSConfig{
		OutputDir:      opts.AudioCacheDir,
		Provider:       provider,
		ProviderConfig: cfg.TTS,
	}, ttsProviders)))

	sttProviders := map[string]tools.TranscriptionProvider{}
	sttProviders["local"] = tools.NewLocalSTTProvider(opts.TranscriptionCacheDir)
	tools.RegisterTranscriptionProviders(sttProviders, tools.TranscriptionProviderConfig{
		APIKey:  os.Getenv("GORMES_STT_OPENAI_KEY"),
		BaseURL: os.Getenv("GORMES_STT_OPENAI_BASE_URL"),
		Model:   "",
	})
	reg.MustRegister(tools.NewTranscriptionTool(tools.NewTranscriptionRunner(ConfiguredTranscriptionConfig(cfg), sttProviders)))
}

func ConfiguredTranscriptionConfig(cfg config.Config) tools.TranscriptionConfig {
	return tools.TranscriptionConfig{
		Disabled:    false,
		Provider:    strings.ToLower(strings.TrimSpace(cfg.STT.Provider)),
		LocalModel:  strings.TrimSpace(cfg.STT.Local.Model),
		OpenAIModel: strings.TrimSpace(cfg.STT.OpenAI.Model),
		Language:    strings.TrimSpace(cfg.STT.Local.Language),
	}
}

func ConfiguredTTSProviderFromMap(ttsConfig map[string]any) string {
	if ttsConfig == nil {
		return ""
	}
	value, ok := ttsConfig["provider"].(string)
	if !ok {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(value))
}
