package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func setupMigrateOpenClawEnv(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("HOME", filepath.Join(root, "fake-home"))
	t.Setenv("GORMES_TELEGRAM_BOT_TOKEN", "")
	t.Setenv("GORMES_DISCORD_BOT_TOKEN", "")
	return root
}

func writeOpenClawCLIFixture(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	cfg := `model: gpt-4.1-mini
providers:
  openrouter:
    api_key:
      source: env
      id: OPENROUTER_API_KEY
channels:
  telegram:
    bot_token:
      source: env
      id: TELEGRAM_BOT_TOKEN
mcp:
  servers:
    - name: notes
ui:
  theme: dark
unknown_top_level_section:
  ignored: true
`
	envBody := `TELEGRAM_BOT_TOKEN=plain-telegram-token
RANDOM_USER_VAR=plainvalue
`
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(envBody), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "MEMORY.md"), []byte("# memory\n"), 0o600); err != nil {
		t.Fatalf("write memory: %v", err)
	}
}

func executeMigrateOpenClaw(args ...string) (*cobra.Command, string, string, error) {
	cmd := newMigrateCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return cmd, stdout.String(), stderr.String(), err
}

// TestMigrateOpenClawDryRun_JSONIncludesBuildProvenance proves
// `gormes migrate openclaw --dry-run` emits a top-level `build`
// envelope so fleet automation orchestrating OpenClaw-to-Gormes
// migration across machines can attribute each manifest to the binary
// version that emitted it. Existing top-level fields stay addressable.
func TestMigrateOpenClawDryRun_JSONIncludesBuildProvenance(t *testing.T) {
	root := setupMigrateOpenClawEnv(t)
	src := filepath.Join(root, "src")
	writeOpenClawCLIFixture(t, src)

	_, stdout, stderr, err := executeMigrateOpenClaw("openclaw", "--dry-run", "--source", src)
	if err != nil {
		t.Fatalf("migrate openclaw --dry-run: %v\nstderr=%s", err, stderr)
	}
	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Source struct {
			Selected string `json:"selected"`
		} `json:"source"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("invalid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Source.Selected != "explicit_source" {
		t.Errorf("source.selected = %q, want explicit_source (still addressable)", got.Source.Selected)
	}
}

func TestMigrateOpenClawDryRun_PrintsManifestJSONAndCounts(t *testing.T) {
	root := setupMigrateOpenClawEnv(t)
	src := filepath.Join(root, "src")
	writeOpenClawCLIFixture(t, src)

	_, stdout, stderr, err := executeMigrateOpenClaw("openclaw", "--dry-run", "--source", src)
	if err != nil {
		t.Fatalf("migrate openclaw --dry-run: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var doc struct {
		Source struct {
			Selected     string `json:"selected"`
			SelectedPath string `json:"selected_path"`
		} `json:"source"`
		Counts struct {
			Migrated int `json:"migrated"`
			Skipped  int `json:"skipped"`
			Archived int `json:"archived"`
			Errors   int `json:"errors"`
		} `json:"counts"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("stdout is not JSON: %v\nstdout=%s", err, stdout)
	}
	if doc.Source.Selected != "explicit_source" || doc.Source.SelectedPath != src {
		t.Fatalf("unexpected source: %+v", doc.Source)
	}
	if doc.Counts.Migrated < 1 || doc.Counts.Skipped < 1 || doc.Counts.Archived < 1 || doc.Counts.Errors != 0 {
		t.Fatalf("unexpected counts: %+v", doc.Counts)
	}
	if strings.Contains(stdout, "plain-telegram-token") {
		t.Fatalf("stdout leaked raw secret: %s", stdout)
	}
}

func TestMigrateOpenClawDryRun_RejectsMissingDryRunAndTypo(t *testing.T) {
	root := setupMigrateOpenClawEnv(t)
	src := filepath.Join(root, "src")
	writeOpenClawCLIFixture(t, src)

	_, _, stderr, err := executeMigrateOpenClaw("openclaw", "--source", src)
	if err == nil || !strings.Contains(err.Error()+stderr, "--dry-run") {
		t.Fatalf("missing --dry-run should fail with dry-run error, err=%v stderr=%s", err, stderr)
	}
	if exitCodeFromError(err) == 0 {
		t.Fatalf("missing --dry-run exit code should be non-zero")
	}

	_, _, stderr, err = executeMigrateOpenClaw("ooenclaw", "--dry-run", "--source", src)
	if err == nil {
		t.Fatalf("ooenclaw should fail")
	}
	if !strings.Contains(err.Error()+stderr, "openclaw") {
		t.Fatalf("expected typo suggestion to mention openclaw, err=%v stderr=%s", err, stderr)
	}
}
