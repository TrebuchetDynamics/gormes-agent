package integration_test

import (
	. "github.com/TrebuchetDynamics/gormes-agent/internal/config"

	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigParsesAgentImageInputModeAndAuxiliaryVision(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	path := filepath.Join(home, "config.toml")
	if err := os.WriteFile(path, []byte(`
[agent]
image_input_mode = " text "

[auxiliary.vision]
provider = " openai "
model = " gpt-4o-mini "
base_url = " https://example.test/v1 "
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Agent.ImageInputMode != "text" {
		t.Fatalf("Agent.ImageInputMode = %q, want text", cfg.Agent.ImageInputMode)
	}
	if cfg.Auxiliary.Vision.Provider != "openai" ||
		cfg.Auxiliary.Vision.Model != "gpt-4o-mini" ||
		cfg.Auxiliary.Vision.BaseURL != "https://example.test/v1" {
		t.Fatalf("Auxiliary.Vision = %#v, want trimmed explicit vision route", cfg.Auxiliary.Vision)
	}
}
