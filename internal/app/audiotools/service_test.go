//go:build !gormes_lite && !slim

package audiotools

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestRegisterAddsTTSAndSTTTools(t *testing.T) {
	reg := tools.NewRegistry()
	Register(reg, config.Config{}, Options{AudioCacheDir: t.TempDir(), TranscriptionCacheDir: t.TempDir(), RuntimeTTSProvider: "edge"})
	if _, ok := reg.Get("text_to_speech"); !ok {
		t.Fatal("text_to_speech tool not registered")
	}
	if _, ok := reg.Get("transcribe_audio"); !ok {
		t.Fatal("transcribe_audio tool not registered")
	}
}

func TestConfiguredTranscriptionConfigTrimsAndNormalizes(t *testing.T) {
	got := ConfiguredTranscriptionConfig(config.Config{STT: config.STTCfg{
		Provider: " LOCAL ",
		Local:    config.STTLocalCfg{Model: " tiny.en ", Language: " en "},
		OpenAI:   config.STTProviderCfg{Model: " gpt-4o-transcribe "},
	}})
	if got.Provider != "local" || got.LocalModel != "tiny.en" || got.Language != "en" || got.OpenAIModel != "gpt-4o-transcribe" {
		t.Fatalf("config = %+v", got)
	}
}

func TestConfiguredTTSProviderFromMap(t *testing.T) {
	if got := ConfiguredTTSProviderFromMap(map[string]any{"provider": " Edge "}); got != "edge" {
		t.Fatalf("provider = %q, want edge", got)
	}
	if got := ConfiguredTTSProviderFromMap(map[string]any{"provider": 12}); got != "" {
		t.Fatalf("provider = %q, want empty for non-string", got)
	}
}
