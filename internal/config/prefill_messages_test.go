package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPrefillMessagesResolvesRelativePathFromGormesHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	if err := os.WriteFile(filepath.Join(home, "prefill.json"), []byte(`[
		{"role":"user","content":"example request"},
		{"role":"assistant","content":"example answer"}
	]`), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	messages, err := LoadPrefillMessages("prefill.json")
	if err != nil {
		t.Fatalf("LoadPrefillMessages: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(messages))
	}
	if messages[0].Role != "user" || messages[0].Content != "example request" {
		t.Fatalf("messages[0] = %#v", messages[0])
	}
	if messages[1].Role != "assistant" || messages[1].Content != "example answer" {
		t.Fatalf("messages[1] = %#v", messages[1])
	}
}

func TestLoadPrefillMessagesMissingOrInvalidReturnsEmpty(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	messages, err := LoadPrefillMessages("missing.json")
	if err != nil {
		t.Fatalf("missing LoadPrefillMessages error = %v, want nil", err)
	}
	if len(messages) != 0 {
		t.Fatalf("missing messages = %#v, want empty", messages)
	}

	invalid := filepath.Join(GormesHome(), "invalid.json")
	if err := os.WriteFile(invalid, []byte(`{"role":"user"}`), 0o600); err != nil {
		t.Fatalf("write invalid: %v", err)
	}
	messages, err = LoadPrefillMessages(invalid)
	if err != nil {
		t.Fatalf("invalid LoadPrefillMessages error = %v, want nil", err)
	}
	if len(messages) != 0 {
		t.Fatalf("invalid messages = %#v, want empty", messages)
	}
}

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
