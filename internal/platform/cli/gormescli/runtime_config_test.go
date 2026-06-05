package gormescli

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestConfiguredMaxToolIterations(t *testing.T) {
	if got := ConfiguredMaxToolIterations(config.Config{}); got != kernel.DefaultMaxToolIterations {
		t.Fatalf("default max tool iterations = %d, want %d", got, kernel.DefaultMaxToolIterations)
	}
	if got := ConfiguredMaxToolIterations(config.Config{Runtime: config.RuntimeCfg{MaxToolIterations: 17}}); got != 17 {
		t.Fatalf("configured max tool iterations = %d, want 17", got)
	}
}

func TestConfiguredTTSProvider(t *testing.T) {
	if got := ConfiguredTTSProvider(config.Config{}); got != "edge" {
		t.Fatalf("default TTS provider = %q, want edge", got)
	}
	if got := ConfiguredTTSProvider(config.Config{Runtime: config.RuntimeCfg{TTSProvider: " OpenAI "}}); got != "openai" {
		t.Fatalf("configured TTS provider = %q, want openai", got)
	}
}
