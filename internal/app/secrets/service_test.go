package secrets

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/externalsecrets"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestBitwardenSyncDryRunDoesNotMutateEnvOrSourceLabels(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(`[secrets.bitwarden]
enabled = true
project_id = "project-123"
access_token_env = "BWS_ACCESS_TOKEN"
override_existing = true
auto_install = false
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	fakeBWS := filepath.Join(home, "bws")
	if err := os.WriteFile(fakeBWS, []byte("#!/bin/sh\nprintf '%s' '[{\"key\":\"GORMES_API_KEY\",\"value\":\"sk-bitwarden-secret\"},{\"key\":\"BWS_ACCESS_TOKEN\",\"value\":\"0.malicious\"}]'\n"), 0o700); err != nil {
		t.Fatalf("write fake bws: %v", err)
	}
	t.Setenv("PATH", home+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("BWS_ACCESS_TOKEN", "0.bootstrap")
	t.Setenv("GORMES_API_KEY", "sk-existing-secret")
	externalsecrets.ResetSecretSourcesForTests()

	var out bytes.Buffer
	if err := BitwardenSync(context.Background(), &out, false); err != nil {
		t.Fatalf("BitwardenSync dry-run: %v\nout=%s", err, out.String())
	}
	if got := os.Getenv("GORMES_API_KEY"); got != "sk-existing-secret" {
		t.Fatalf("dry-run mutated env: %q", got)
	}
	if got := externalsecrets.GetSecretSource("GORMES_API_KEY"); got != "" {
		t.Fatalf("dry-run recorded source label: %q", got)
	}
	if strings.Contains(out.String(), "sk-bitwarden-secret") || !strings.Contains(out.String(), "skip (already set)") || !strings.Contains(out.String(), "skip (bootstrap token)") {
		t.Fatalf("unexpected dry-run output:\n%s", out.String())
	}
}

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
