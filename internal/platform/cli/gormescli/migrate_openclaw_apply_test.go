package gormescli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func writeOpenClawApplyFixture(t *testing.T, dir string) {
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
ui:
  theme: dark
`
	envBody := `TELEGRAM_BOT_TOKEN=plain-telegram-token
OPENROUTER_API_KEY=plain-openrouter-key
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

// TestMigrateOpenClawApply_JSONIncludesBuildProvenance proves
// `gormes migrate openclaw --yes` emits a top-level `build` envelope so
// fleet automation orchestrating OpenClaw-to-Gormes migration across
// machines can attribute each apply outcome to the binary version that
// emitted it. Existing top-level fields stay addressable.
func TestMigrateOpenClawApply_JSONIncludesBuildProvenance(t *testing.T) {
	root := setupMigrateOpenClawEnv(t)
	src := filepath.Join(root, "openclaw-src")
	writeOpenClawApplyFixture(t, src)
	dest := filepath.Join(root, "dest")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}
	_, stdout, stderr, err := executeMigrateOpenClaw(
		"openclaw", "--yes",
		"--source", src,
		"--dest", dest,
		"--secrets",
	)
	if err != nil {
		t.Fatalf("migrate openclaw --yes: %v\nstderr=%s", err, stderr)
	}
	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Counts struct {
			Migrated int `json:"migrated"`
		} `json:"counts"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("invalid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Counts.Migrated < 1 {
		t.Errorf("counts.migrated = %d, want >= 1 (still addressable)", got.Counts.Migrated)
	}
}

func TestMigrateOpenClawApply_CommandWiring(t *testing.T) {
	root := setupMigrateOpenClawEnv(t)
	src := filepath.Join(root, "openclaw-src")
	writeOpenClawApplyFixture(t, src)
	dest := filepath.Join(root, "dest")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}

	_, stdout, stderr, err := executeMigrateOpenClaw(
		"openclaw", "--yes",
		"--source", src,
		"--dest", dest,
		"--secrets",
	)
	if err != nil {
		t.Fatalf("migrate openclaw --yes: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	var doc struct {
		ConfigWritten map[string]string `json:"config_written"`
		EnvWritten    map[string]string `json:"env_written"`
		MemoryWritten map[string]string `json:"memory_written"`
		SkillWritten  map[string]string `json:"skill_written"`
		ReportPath    string            `json:"report_path"`
		Counts        struct {
			Migrated        int `json:"migrated"`
			Skipped         int `json:"skipped"`
			ConflictSkipped int `json:"conflict_skipped"`
			Archived        int `json:"archived"`
			SecretSkipped   int `json:"secret_skipped"`
			Errors          int `json:"errors"`
		} `json:"counts"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("stdout is not JSON: %v\nstdout=%s", err, stdout)
	}
	if doc.Counts.Migrated < 1 {
		t.Fatalf("expected migrated >= 1, got %+v\nstdout=%s", doc.Counts, stdout)
	}
	if doc.ReportPath == "" {
		t.Fatalf("expected report_path in stdout, got %s", stdout)
	}
	// stdout must not leak raw secret bytes.
	for _, leak := range []string{"plain-telegram-token", "plain-openrouter-key"} {
		if strings.Contains(stdout, leak) {
			t.Fatalf("stdout leaked secret bytes %q: %s", leak, stdout)
		}
	}
	// dest config.toml must exist.
	if _, statErr := os.Stat(filepath.Join(dest, "config.toml")); statErr != nil {
		t.Fatalf("dest/config.toml missing after --yes: %v", statErr)
	}
}

// TestMigrateOpenClawCleanup_JSONIncludesBuildProvenance proves
// `gormes migrate openclaw cleanup --dry-run` emits a top-level
// `build` envelope so fleet automation orchestrating
// post-OpenClaw-migration archival across machines can attribute each
// cleanup outcome to the binary version that emitted it. Existing
// top-level fields stay addressable.
func TestMigrateOpenClawCleanup_JSONIncludesBuildProvenance(t *testing.T) {
	root := setupMigrateOpenClawEnv(t)
	home := filepath.Join(root, "fake-home")
	for _, dir := range []string{".openclaw", ".clawdbot", ".moltbot"} {
		if err := os.MkdirAll(filepath.Join(home, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	_, stdout, stderr, err := executeMigrateOpenClaw("openclaw", "cleanup", "--dry-run")
	if err != nil {
		t.Fatalf("migrate openclaw cleanup --dry-run: %v\nstderr=%s", err, stderr)
	}
	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		DryRun bool `json:"dry_run"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("invalid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != Version {
		t.Errorf("build.version = %q, want %q", got.Build.Version, Version)
	}
	if !got.DryRun {
		t.Errorf("dry_run = false, want true (still addressable)")
	}
}

func TestMigrateOpenClawCleanup_DryRunCommandWiring(t *testing.T) {
	root := setupMigrateOpenClawEnv(t)
	home := filepath.Join(root, "fake-home")
	for _, dir := range []string{".openclaw", ".clawdbot", ".moltbot"} {
		if err := os.MkdirAll(filepath.Join(home, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	_, stdout, stderr, err := executeMigrateOpenClaw("openclaw", "cleanup", "--dry-run")
	if err != nil {
		t.Fatalf("migrate openclaw cleanup --dry-run: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var doc struct {
		Renamed []struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"renamed"`
		DryRun bool `json:"dry_run"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("stdout is not JSON: %v\nstdout=%s", err, stdout)
	}
	if !doc.DryRun {
		t.Fatalf("expected dry_run=true; got %+v", doc)
	}
	if len(doc.Renamed) != 3 {
		t.Fatalf("expected 3 renames previewed, got %d: %+v", len(doc.Renamed), doc.Renamed)
	}
	// dirs untouched.
	for _, dir := range []string{".openclaw", ".clawdbot", ".moltbot"} {
		if _, err := os.Stat(filepath.Join(home, dir)); err != nil {
			t.Fatalf("dry-run cleanup should leave %s untouched: %v", dir, err)
		}
	}
}

func newClawRootCommandForTest() *cobra.Command {
	return newRootCommandWithFactoryForTest("claw", func() *cobra.Command {
		return NewClawCommand(MigrateCommandOptions{BuildProvenance: testBuildProvenance, ExitCodeError: NewExitCodeError})
	})
}

func TestClawCleanupDryRunCompatibilityJSON(t *testing.T) {
	root := setupMigrateOpenClawEnv(t)
	home := filepath.Join(root, "fake-home")
	for _, dir := range []string{".openclaw", ".clawdbot", ".moltbot"} {
		if err := os.MkdirAll(filepath.Join(home, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	for _, args := range [][]string{
		{"claw", "cleanup", "--dry-run"},
		{"claw", "clean", "--dry-run"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			stdout, stderr, err := executeRootCommandForTest(newClawRootCommandForTest(), args...)
			if err != nil {
				t.Fatalf("%s: %v\nstdout=%s\nstderr=%s", strings.Join(args, " "), err, stdout, stderr)
			}
			var doc struct {
				Build struct {
					Version string `json:"version"`
				} `json:"build"`
				Renamed []struct {
					From string `json:"from"`
					To   string `json:"to"`
				} `json:"renamed"`
				DryRun bool `json:"dry_run"`
			}
			if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
				t.Fatalf("stdout is not JSON: %v\nstdout=%s", err, stdout)
			}
			if doc.Build.Version != Version {
				t.Errorf("build.version = %q, want %q", doc.Build.Version, Version)
			}
			if !doc.DryRun {
				t.Fatalf("dry_run = false, want true: %+v", doc)
			}
			if len(doc.Renamed) != 3 {
				t.Fatalf("expected 3 renames previewed, got %d: %+v", len(doc.Renamed), doc.Renamed)
			}
		})
	}
}
