package integration_test

import (
	. "github.com/TrebuchetDynamics/gormes-agent/internal/config"

	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_GonchoDefaults(t *testing.T) {
	isolateGonchoConfig(t)

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}

	if !cfg.Goncho.Enabled || cfg.Goncho.Workspace != "gormes" || cfg.Goncho.DialecticDefaultLevel != "low" {
		t.Fatalf("Goncho defaults = %+v, want enabled gormes/low defaults", cfg.Goncho)
	}
}

func TestLoad_GonchoEnvOverridesFile(t *testing.T) {
	cfgHome := isolateGonchoConfig(t)
	writeGonchoConfigFile(t, cfgHome, `
[goncho]
enabled = true
workspace = "file-workspace"
observer_peer = "file-observer"
recent_messages = 6
max_message_size = 111
max_file_size = 222
get_context_max_tokens = 333
reasoning_enabled = true
peer_card_enabled = true
summary_enabled = true
dream_enabled = false
deriver_workers = 2
representation_batch_max_tokens = 444
dialectic_default_level = "minimal"
`)

	t.Setenv("GORMES_GONCHO_ENABLED", "false")
	t.Setenv("GORMES_GONCHO_WORKSPACE", "env-workspace")
	t.Setenv("GORMES_GONCHO_OBSERVER_PEER", "env-observer")
	t.Setenv("GORMES_GONCHO_RECENT_MESSAGES", "7")
	t.Setenv("GORMES_GONCHO_MAX_MESSAGE_SIZE", "25001")
	t.Setenv("GORMES_GONCHO_MAX_FILE_SIZE", "5242881")
	t.Setenv("GORMES_GONCHO_GET_CONTEXT_MAX_TOKENS", "99999")
	t.Setenv("GORMES_GONCHO_REASONING_ENABLED", "false")
	t.Setenv("GORMES_GONCHO_PEER_CARD_ENABLED", "false")
	t.Setenv("GORMES_GONCHO_SUMMARY_ENABLED", "false")
	t.Setenv("GORMES_GONCHO_DREAM_ENABLED", "true")
	t.Setenv("GORMES_GONCHO_DERIVER_WORKERS", "3")
	t.Setenv("GORMES_GONCHO_REPRESENTATION_BATCH_MAX_TOKENS", "2048")
	t.Setenv("GORMES_GONCHO_DIALECTIC_DEFAULT_LEVEL", "high")

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Goncho.Enabled {
		t.Error("Goncho.Enabled = true, want env false")
	}
	if cfg.Goncho.Workspace != "env-workspace" {
		t.Errorf("Goncho.Workspace = %q, want env-workspace", cfg.Goncho.Workspace)
	}
	if cfg.Goncho.ObserverPeer != "env-observer" {
		t.Errorf("Goncho.ObserverPeer = %q, want env-observer", cfg.Goncho.ObserverPeer)
	}
	if cfg.Goncho.RecentMessages != 7 {
		t.Errorf("Goncho.RecentMessages = %d, want 7", cfg.Goncho.RecentMessages)
	}
	if cfg.Goncho.MaxMessageSize != 25_001 {
		t.Errorf("Goncho.MaxMessageSize = %d, want 25001", cfg.Goncho.MaxMessageSize)
	}
	if cfg.Goncho.MaxFileSize != 5_242_881 {
		t.Errorf("Goncho.MaxFileSize = %d, want 5242881", cfg.Goncho.MaxFileSize)
	}
	if cfg.Goncho.GetContextMaxTokens != 99_999 {
		t.Errorf("Goncho.GetContextMaxTokens = %d, want 99999", cfg.Goncho.GetContextMaxTokens)
	}
	if cfg.Goncho.ReasoningEnabled {
		t.Error("Goncho.ReasoningEnabled = true, want env false")
	}
	if cfg.Goncho.PeerCardEnabled {
		t.Error("Goncho.PeerCardEnabled = true, want env false")
	}
	if cfg.Goncho.SummaryEnabled {
		t.Error("Goncho.SummaryEnabled = true, want env false")
	}
	if !cfg.Goncho.DreamEnabled {
		t.Error("Goncho.DreamEnabled = false, want env true")
	}
	if cfg.Goncho.DeriverWorkers != 3 {
		t.Errorf("Goncho.DeriverWorkers = %d, want 3", cfg.Goncho.DeriverWorkers)
	}
	if cfg.Goncho.RepresentationBatchMaxTokens != 2048 {
		t.Errorf("Goncho.RepresentationBatchMaxTokens = %d, want 2048", cfg.Goncho.RepresentationBatchMaxTokens)
	}
	if cfg.Goncho.DialecticDefaultLevel != "high" {
		t.Errorf("Goncho.DialecticDefaultLevel = %q, want high", cfg.Goncho.DialecticDefaultLevel)
	}
}

func TestLoad_GonchoRejectsInvalidDialecticDefaultLevel(t *testing.T) {
	cfgHome := isolateGonchoConfig(t)
	writeGonchoConfigFile(t, cfgHome, `
[goncho]
dialectic_default_level = "extreme"
`)

	_, err := Load(nil)
	if err == nil {
		t.Fatal("Load() error = nil, want invalid dialectic_default_level error")
	}
	if !strings.Contains(err.Error(), "goncho.dialectic_default_level") {
		t.Fatalf("Load() error = %v, want goncho.dialectic_default_level", err)
	}
}

func TestLoad_GonchoRejectsNegativeLimits(t *testing.T) {
	cfgHome := isolateGonchoConfig(t)
	writeGonchoConfigFile(t, cfgHome, "\n[goncho]\nrecent_messages = -1\n")

	_, err := Load(nil)
	if err == nil {
		t.Fatal("Load() error = nil, want negative limit error")
	}
	if !strings.Contains(err.Error(), "goncho.recent_messages") {
		t.Fatalf("Load() error = %v, want goncho.recent_messages", err)
	}
}

func isolateGonchoConfig(t *testing.T) string {
	t.Helper()
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", filepath.Join(cfgHome, "gormes"))
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("HERMES_HOME", "")
	for _, key := range []string{
		"GORMES_GONCHO_ENABLED",
		"GORMES_GONCHO_WORKSPACE",
		"GORMES_GONCHO_OBSERVER_PEER",
		"GORMES_GONCHO_RECENT_MESSAGES",
		"GORMES_GONCHO_MAX_MESSAGE_SIZE",
		"GORMES_GONCHO_MAX_FILE_SIZE",
		"GORMES_GONCHO_GET_CONTEXT_MAX_TOKENS",
		"GORMES_GONCHO_REASONING_ENABLED",
		"GORMES_GONCHO_PEER_CARD_ENABLED",
		"GORMES_GONCHO_SUMMARY_ENABLED",
		"GORMES_GONCHO_DREAM_ENABLED",
		"GORMES_GONCHO_DREAM_IDLE_TIMEOUT_MINUTES",
		"GORMES_GONCHO_DERIVER_WORKERS",
		"GORMES_GONCHO_REPRESENTATION_BATCH_MAX_TOKENS",
		"GORMES_GONCHO_DIALECTIC_DEFAULT_LEVEL",
	} {
		t.Setenv(key, "")
	}
	return cfgHome
}

func writeGonchoConfigFile(t *testing.T, cfgHome, body string) {
	t.Helper()
	dir := filepath.Join(cfgHome, "gormes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
