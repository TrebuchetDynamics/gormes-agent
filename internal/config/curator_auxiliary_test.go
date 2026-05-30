package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCuratorAuxiliarySlot_DefaultConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GORMES_HOME", filepath.Join(t.TempDir(), "gormes"))
	t.Setenv("GORMES_ENDPOINT", "")
	t.Setenv("GORMES_API_KEY", "")

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}

	slot := cfg.Auxiliary.Curator
	if slot.Provider != "auto" || slot.Model != "" || slot.BaseURL != "" || slot.APIKey != "" || slot.Timeout < 600 || slot.ExtraBody == nil {
		t.Fatalf("Auxiliary.Curator default route = %+v, want safe auto default", slot)
	}
}

func TestCuratorAuxiliarySlot_LoadsCanonicalAndLegacyTOML(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GORMES_HOME", home)
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(`
[hermes]
provider = "openrouter"
model = "openai/gpt-5.5"

[auxiliary.curator]
provider = "custom"
model = "local-mini"
base_url = "http://localhost:11434/v1"
api_key = "sk-curator-only"
timeout = 900

[auxiliary.curator.extra_body]
reasoning_effort = "low"

[curator.auxiliary]
provider = "openrouter"
model = "legacy-model"
api_key = "legacy-secret"
base_url = "http://legacy/v1"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}

	slot := cfg.Auxiliary.Curator
	if slot.Provider != "custom" || slot.Model != "local-mini" || slot.BaseURL != "http://localhost:11434/v1" || slot.APIKey != "sk-curator-only" || slot.Timeout != 900 {
		t.Fatalf("Auxiliary.Curator = %#v, want canonical custom/local-mini slot", slot)
	}
	if got := slot.ExtraBody["reasoning_effort"]; got != "low" {
		t.Fatalf("Auxiliary.Curator.ExtraBody[reasoning_effort] = %#v, want low", got)
	}
	legacy := cfg.Curator.Auxiliary
	if legacy.Provider != "openrouter" || legacy.Model != "legacy-model" || legacy.APIKey != "legacy-secret" || legacy.BaseURL != "http://legacy/v1" {
		t.Fatalf("Curator.Auxiliary legacy slot = %#v", legacy)
	}
}
