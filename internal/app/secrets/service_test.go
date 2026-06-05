package secrets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestWriteSnapshotFilePreservesSymlink(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	realPath := filepath.Join(home, "real-secrets-runtime.json")
	if err := os.WriteFile(realPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write real snapshot: %v", err)
	}
	if err := os.Symlink(realPath, filepath.Join(home, "secrets-runtime.json")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	err := WriteSnapshotFile(toolspkg.SecretsRuntimeSnapshot{
		Entries: map[string]toolspkg.SecretsRuntimeEntry{
			"hermes.api_key": {
				Path:     "hermes.api_key",
				Ref:      toolspkg.SecretRef{Source: toolspkg.SecretRefSourceEnv, Provider: "default", ID: "GORMES_API_KEY"},
				Resolved: true,
				Evidence: toolspkg.SecretRefEvidence{Redacted: true},
			},
		},
	})
	if err != nil {
		t.Fatalf("WriteSnapshotFile: %v", err)
	}

	info, err := os.Lstat(filepath.Join(home, "secrets-runtime.json"))
	if err != nil {
		t.Fatalf("lstat snapshot link: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("snapshot link was replaced with mode %v, want symlink preserved", info.Mode())
	}
	got, err := os.ReadFile(realPath)
	if err != nil {
		t.Fatalf("read real snapshot: %v", err)
	}
	if !strings.Contains(string(got), "GORMES_API_KEY") {
		t.Fatalf("real snapshot was not updated through symlink:\n%s", got)
	}
}
