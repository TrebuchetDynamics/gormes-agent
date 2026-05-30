package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestYuanbaoConfig_DisabledByDefault(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Yuanbao.Enabled {
		t.Fatalf("Yuanbao.Enabled = true, want false (disabled by default)")
	}
	if cfg.Yuanbao.RuntimeEnabled() {
		t.Fatalf("Yuanbao.RuntimeEnabled() = true, want false (no creds)")
	}
}

func TestYuanbaoConfig_LoadParsesDisabledByDefaultSectionFromTOML(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	dir := filepath.Join(cfgHome, "gormes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`
[yuanbao]
login_token = "fake-login-token"
hy_source = "fake-hy-source"
agent_id = "fake-agent"
allowed_conversation_id = "conv-1"
coalesce_ms = 750
first_run_discovery = true
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Yuanbao.Enabled {
		t.Fatalf("Yuanbao.Enabled = true, want false (omitted from TOML)")
	}
	if cfg.Yuanbao.LoginToken != "fake-login-token" {
		t.Fatalf("Yuanbao.LoginToken = %q", cfg.Yuanbao.LoginToken)
	}
	if cfg.Yuanbao.HySource != "fake-hy-source" || cfg.Yuanbao.AgentID != "fake-agent" {
		t.Fatalf("Yuanbao identity fields = %#v", cfg.Yuanbao)
	}
	if cfg.Yuanbao.AllowedConversationID != "conv-1" {
		t.Fatalf("Yuanbao.AllowedConversationID = %q", cfg.Yuanbao.AllowedConversationID)
	}
	if cfg.Yuanbao.CoalesceMs != 750 || !cfg.Yuanbao.FirstRunDiscovery {
		t.Fatalf("Yuanbao knobs = %#v", cfg.Yuanbao)
	}
	if cfg.Yuanbao.RuntimeEnabled() {
		t.Fatalf("RuntimeEnabled() = true, want false (Enabled flag missing)")
	}
}
