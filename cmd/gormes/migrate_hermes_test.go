package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func setupMigrateHermesEnv(t *testing.T) (root string) {
	t.Helper()
	root = t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("HERMES_HOME", "")
	t.Setenv("HOME", filepath.Join(root, "fake-home"))
	t.Setenv("GORMES_TELEGRAM_TOKEN", "")
	t.Setenv("GORMES_DISCORD_TOKEN", "")
	return root
}

func writeMigrateFixture(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	cfg := `model: gpt-4.1-mini
providers:
  openrouter:
    api_key: sk-secret-redacted-in-env
custom_provider:
  base_url: https://example.invalid/v1
terminal:
  backend: local
display:
  theme: dark
gateway:
  notify_interval: 180
memory:
  enabled: true
unknown_top_level_section:
  ignored: true
`
	envBody := `TELEGRAM_TOKEN=tg-secret
DISCORD_TOKEN=dc-secret
RANDOM_USER_VAR=plainvalue
`
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(envBody), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
}

func snapshotMigrateDir(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	_ = filepath.Walk(root, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		out = append(out, path)
		return nil
	})
	sort.Strings(out)
	return out
}

func executeMigrateHermes(args ...string) (*cobra.Command, string, string, error) {
	cmd := newMigrateCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return cmd, stdout.String(), stderr.String(), err
}

func TestMigrateHermesDryRun_PrintsManifestJSONAndCounts(t *testing.T) {
	root := setupMigrateHermesEnv(t)
	src := filepath.Join(root, "src")
	writeMigrateFixture(t, src)

	_, stdout, stderr, err := executeMigrateHermes("hermes", "--dry-run", "--source", src)
	if err != nil {
		t.Fatalf("migrate hermes --dry-run: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	var doc struct {
		Source struct {
			Selected     string `json:"selected"`
			SelectedPath string `json:"selected_path"`
		} `json:"source"`
		Counts struct {
			Migrated int `json:"migrated"`
			Skipped  int `json:"skipped"`
			Conflict int `json:"conflict"`
			Errors   int `json:"errors"`
		} `json:"counts"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("stdout is not JSON: %v\nstdout=%s", err, stdout)
	}
	if doc.Source.Selected != "explicit_source" {
		t.Fatalf("Selected = %q, want explicit_source", doc.Source.Selected)
	}
	if doc.Source.SelectedPath != src {
		t.Fatalf("SelectedPath = %q, want %q", doc.Source.SelectedPath, src)
	}
	if doc.Counts.Migrated < 1 {
		t.Fatalf("expected migrated >= 1, got %+v", doc.Counts)
	}
	if doc.Counts.Skipped < 1 {
		t.Fatalf("expected skipped >= 1 (unknown_top_level_section + RANDOM_USER_VAR), got %+v", doc.Counts)
	}
	if strings.Contains(stdout, "tg-secret") || strings.Contains(stdout, "dc-secret") || strings.Contains(stdout, "sk-secret") {
		t.Fatalf("stdout leaked raw secrets: %s", stdout)
	}
}

func TestMigrateHermesDryRun_RejectsMissingSourceWithNonZeroExit(t *testing.T) {
	root := setupMigrateHermesEnv(t)
	missing := filepath.Join(root, "no-such-source")

	_, _, _, err := executeMigrateHermes("hermes", "--dry-run", "--source", missing)
	if err == nil {
		t.Fatalf("missing source should return non-zero exit, got nil")
	}
	if exitCodeFromError(err) == 0 {
		t.Fatalf("missing source exit code = 0, want non-zero")
	}
}

func TestMigrateHermesDryRun_DoesNotMutateXDGRoots(t *testing.T) {
	root := setupMigrateHermesEnv(t)
	src := filepath.Join(root, "src")
	writeMigrateFixture(t, src)

	cfg := os.Getenv("XDG_CONFIG_HOME")
	data := os.Getenv("XDG_DATA_HOME")
	state := os.Getenv("XDG_STATE_HOME")
	for _, d := range []string{cfg, data, state} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	beforeCfg := snapshotMigrateDir(t, cfg)
	beforeData := snapshotMigrateDir(t, data)
	beforeState := snapshotMigrateDir(t, state)

	if _, _, _, err := executeMigrateHermes("hermes", "--dry-run", "--source", src); err != nil {
		t.Fatalf("migrate hermes --dry-run: %v", err)
	}

	if got := snapshotMigrateDir(t, cfg); !sliceEqual(got, beforeCfg) {
		t.Fatalf("XDG_CONFIG_HOME mutated: before=%v after=%v", beforeCfg, got)
	}
	if got := snapshotMigrateDir(t, data); !sliceEqual(got, beforeData) {
		t.Fatalf("XDG_DATA_HOME mutated: before=%v after=%v", beforeData, got)
	}
	if got := snapshotMigrateDir(t, state); !sliceEqual(got, beforeState) {
		t.Fatalf("XDG_STATE_HOME mutated: before=%v after=%v", beforeState, got)
	}
}

func TestMigrateHermesDryRun_RequiresDryRunFlagInThisSlice(t *testing.T) {
	root := setupMigrateHermesEnv(t)
	src := filepath.Join(root, "src")
	writeMigrateFixture(t, src)

	_, _, stderr, err := executeMigrateHermes("hermes", "--source", src)
	if err == nil {
		t.Fatalf("missing --dry-run should fail in this slice; stderr=%s", stderr)
	}
	if !strings.Contains(err.Error()+stderr, "dry-run") {
		t.Fatalf("error must mention --dry-run, got err=%v stderr=%s", err, stderr)
	}
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
