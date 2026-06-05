//go:build !gormes_lite && !slim

package gormescli

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestRegisterAudioToolsAppliesSTTConfig(t *testing.T) {
	cfg := config.Config{
		STT: config.STTCfg{
			Provider: " openai ",
			Local: config.STTLocalCfg{
				Model:    "tiny.en",
				Language: "es",
			},
			OpenAI: config.STTProviderCfg{Model: "gpt-4o-transcribe"},
		},
	}

	got := configuredTranscriptionConfig(cfg)
	if got.Provider != "openai" {
		t.Fatalf("Provider = %q, want openai", got.Provider)
	}
	if got.LocalModel != "tiny.en" {
		t.Fatalf("LocalModel = %q, want tiny.en", got.LocalModel)
	}
	if got.OpenAIModel != "gpt-4o-transcribe" {
		t.Fatalf("OpenAIModel = %q, want gpt-4o-transcribe", got.OpenAIModel)
	}
	if got.Language != "es" {
		t.Fatalf("Language = %q, want es", got.Language)
	}
	if got.Disabled {
		t.Fatal("Disabled = true, want false because zero-value stt.enabled is not a runtime disable flag")
	}
}
