package gormescli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func setupOneshotFlagTestEnv(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("GORMES_HOME", filepath.Join(root, "gormes"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("HERMES_HOME", filepath.Join(root, "hermes"))
	t.Setenv("CODEX_HOME", filepath.Join(root, "codex"))
	t.Setenv("GORMES_ENDPOINT", "")
	t.Setenv("GORMES_MODEL", "")
	t.Setenv("GORMES_API_KEY", "")
	t.Setenv("GORMES_INFERENCE_MODEL", "")
	t.Setenv("GORMES_INFERENCE_PROVIDER", "")
	t.Setenv("GORMES_KANBAN_DB", "")
	t.Setenv("GORMES_KANBAN_HOME", "")
	t.Setenv("GORMES_KANBAN_TASK", "")
	t.Setenv("HERMES_KANBAN_BOARD", "")
	t.Setenv("HERMES_KANBAN_DB", "")
}

func writeOneshotFlagConfig(t *testing.T, data []byte) {
	t.Helper()
	path := config.ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func executeUpdateCommandForTest(cmd *cobra.Command, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}
