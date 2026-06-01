package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadHermesToolProgressEnvFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
	t.Setenv("HERMES_HOME", filepath.Join(home, "hermes"))
	t.Setenv("HERMES_TOOL_PROGRESS", "false")
	t.Setenv("HERMES_TOOL_PROGRESS_MODE", "")

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load(nil): %v", err)
	}
	if cfg.Display.ToolProgress != "off" {
		t.Fatalf("Display.ToolProgress = %q, want env fallback off", cfg.Display.ToolProgress)
	}
}

func TestLoadHermesToolProgressModeEnvFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
	t.Setenv("HERMES_HOME", filepath.Join(home, "hermes"))
	t.Setenv("HERMES_TOOL_PROGRESS", "")
	t.Setenv("HERMES_TOOL_PROGRESS_MODE", "verbose")

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load(nil): %v", err)
	}
	if cfg.Display.ToolProgress != "verbose" {
		t.Fatalf("Display.ToolProgress = %q, want env fallback verbose", cfg.Display.ToolProgress)
	}
}

func TestLoadHermesToolProgressEnvDoesNotOverrideConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
	t.Setenv("HERMES_HOME", filepath.Join(home, "hermes"))
	t.Setenv("HERMES_TOOL_PROGRESS", "false")
	t.Setenv("HERMES_TOOL_PROGRESS_MODE", "verbose")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(ConfigPath(), []byte("[display]\ntool_progress = 'new'\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load(nil): %v", err)
	}
	if cfg.Display.ToolProgress != "new" {
		t.Fatalf("Display.ToolProgress = %q, want config value to win over deprecated Hermes env", cfg.Display.ToolProgress)
	}
}
