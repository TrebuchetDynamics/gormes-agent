package gormescli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestDoctorOfflineReportsResolvedProviderDefaultModel(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GORMES_HOME", filepath.Join(root, "gormes"))
	t.Setenv("CODEX_HOME", filepath.Join(root, "codex"))
	t.Setenv("GORMES_ENDPOINT", "")
	t.Setenv("GORMES_MODEL", "")
	t.Setenv("GORMES_API_KEY", "")
	t.Setenv("GITHUB_TOKEN", "test-token")
	if err := os.MkdirAll(config.GormesHome(), 0o755); err != nil {
		t.Fatalf("create GORMES_HOME: %v", err)
	}
	if err := os.MkdirAll(os.Getenv("CODEX_HOME"), 0o755); err != nil {
		t.Fatalf("create CODEX_HOME: %v", err)
	}
	if err := os.WriteFile(config.ConfigPath(), []byte("[hermes]\nprovider = 'openai-codex'\nmodel = 'hermes-agent'\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	stdout, stderr, err := executeCobraCommandForTest(newRootCommand(), cobraCommandExecutionOptions{}, "doctor", "--offline")
	if err != nil {
		t.Fatalf("doctor --offline: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	combined := stdout + stderr
	if !strings.Contains(combined, "model=gpt-5.5 source=curated_fallback") {
		t.Fatalf("doctor output missing resolved model provenance:\n%s", combined)
	}
}
