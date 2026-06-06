package gormescmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func setupGatewayStatusTestEnv(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("GORMES_HOME", filepath.Join(root, "gormes"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("HERMES_HOME", filepath.Join(root, "hermes"))
}

func writeGatewayStatusConfig(t *testing.T, data []byte) {
	t.Helper()
	path := config.ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

type gatewayStatusJSON struct {
	Runtime struct {
		Platforms map[string]any `json:"platforms"`
	} `json:"runtime"`
}

func executeGatewayStatusCommand(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	cmd := newRootCommand()
	return executeRootCommandForTest(cmd, append([]string{"gateway", "status"}, args...)...)
}

func assertGatewayStatusDidNotOpenRuntimeStores(t *testing.T) {
	t.Helper()
	for _, path := range []string{
		config.SessionDBPath(),
		config.MemoryDBPath(),
	} {
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("gateway status opened runtime store %s", path)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat runtime store %s: %v", path, err)
		}
	}
}
