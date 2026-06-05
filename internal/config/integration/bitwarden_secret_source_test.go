package integration

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	config "github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config/externalsecrets"
)

func TestConfigLoadAppliesBitwardenSecretsBeforeEnvResolution(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture uses POSIX sh")
	}
	externalsecrets.ResetSecretSourcesForTests()
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_API_KEY", "stale-shell")
	bin := t.TempDir()
	fakeBWS := filepath.Join(bin, "bws")
	if err := os.WriteFile(fakeBWS, []byte("#!/bin/sh\nprintf '%s' '[{\"key\":\"GORMES_API_KEY\",\"value\":\"sk-bitwarden\"}]'\n"), 0o755); err != nil {
		t.Fatalf("write fake bws: %v", err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := os.WriteFile(filepath.Join(home, ".env"), []byte("BWS_ACCESS_TOKEN=0.synthetic\n"), 0o600); err != nil {
		t.Fatalf("write dotenv: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(`
[secrets.bitwarden]
enabled = true
project_id = "project-123"
auto_install = false
override_existing = true
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Hermes.APIKey != "sk-bitwarden" {
		t.Fatalf("Hermes.APIKey = %q, want Bitwarden value", cfg.Hermes.APIKey)
	}
	if got := os.Getenv("BWS_ACCESS_TOKEN"); got != "0.synthetic" {
		t.Fatalf("BWS_ACCESS_TOKEN = %q, want preserved bootstrap token", got)
	}
	if got := externalsecrets.GetSecretSource("GORMES_API_KEY"); got != externalsecrets.BitwardenSourceLabel {
		t.Fatalf("secret source = %q, want bitwarden", got)
	}
}
