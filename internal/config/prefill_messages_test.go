package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPrefillMessagesEnvOverridesConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	configPath := filepath.Join(home, "config-prefill.json")
	envPath := filepath.Join(home, "env-prefill.json")
	if err := os.WriteFile(configPath, []byte(`[{"role":"user","content":"config"}]`), 0o600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}
	if err := os.WriteFile(envPath, []byte(`[{"role":"user","content":"env"}]`), 0o600); err != nil {
		t.Fatalf("write env fixture: %v", err)
	}
	if err := os.WriteFile(ConfigPath(), []byte("[agent]\nprefill_messages_file = \""+configPath+"\"\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("HERMES_PREFILL_MESSAGES_FILE", envPath)

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	messages, err := LoadConfiguredPrefillMessages(cfg)
	if err != nil {
		t.Fatalf("LoadConfiguredPrefillMessages: %v", err)
	}
	if len(messages) != 1 || messages[0].Content != "env" {
		t.Fatalf("messages = %#v, want env override", messages)
	}
}
