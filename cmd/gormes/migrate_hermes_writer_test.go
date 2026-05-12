package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupMigrateHermesWriterEnv wires temp XDG roots for the cobra-level
// writer tests so neither real ~/.gormes nor real XDG paths are touched.
func setupMigrateHermesWriterEnv(t *testing.T) (root string) {
	t.Helper()
	root = t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("HERMES_HOME", "")
	t.Setenv("HOME", filepath.Join(root, "fake-home"))
	t.Setenv("GORMES_TELEGRAM_TOKEN", "")
	t.Setenv("GORMES_TELEGRAM_CHAT_ID", "")
	t.Setenv("GORMES_TELEGRAM_ALLOWED_USERS", "")
	t.Setenv("GORMES_DISCORD_TOKEN", "")
	return root
}

func writeMigrateWriterFixture(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	cfg := `model: gpt-4.1-mini
gateway:
  notify_interval: 180
unknown_section:
  ignored: true
`
	envBody := `TELEGRAM_BOT_TOKEN=tg-secret
TELEGRAM_HOME_CHANNEL=6586915095
TELEGRAM_ALLOWED_USERS=6586915095
ANTHROPIC_API_KEY=sk-deadbeef
RANDOM_USER_VAR=plainvalue
`
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(envBody), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
}

// TestMigrateHermesWriter_JSONIncludesBuildProvenance proves
// `gormes migrate hermes --yes` emits a top-level `build` envelope so
// fleet automation orchestrating Hermes-to-Gormes migration across
// machines can attribute each apply outcome to the binary version that
// emitted it. Existing top-level fields stay addressable via struct
// embedding — additive change.
func TestMigrateHermesWriter_JSONIncludesBuildProvenance(t *testing.T) {
	root := setupMigrateHermesWriterEnv(t)
	src := filepath.Join(root, "hermes-src")
	writeMigrateWriterFixture(t, src)
	dest := filepath.Join(root, "dest")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}

	_, stdout, stderr, err := executeMigrateHermes("hermes", "--yes", "--source", src, "--dest", dest)
	if err != nil {
		t.Fatalf("migrate hermes --yes: %v\nstderr=%s", err, stderr)
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

func TestMigrateHermesWriter_UsesExplicitSource(t *testing.T) {
	root := setupMigrateHermesWriterEnv(t)
	src := filepath.Join(root, "hermes-src")
	writeMigrateWriterFixture(t, src)
	dest := filepath.Join(root, "dest")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}

	_, stdout, stderr, err := executeMigrateHermes("hermes", "--yes", "--source", src, "--dest", dest)
	if err != nil {
		t.Fatalf("migrate hermes --yes: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	var doc struct {
		ConfigWritten map[string]string `json:"config_written"`
		EnvWritten    map[string]string `json:"env_written"`
		Backups       []string          `json:"backups"`
		Counts        struct {
			Migrated        int `json:"migrated"`
			Skipped         int `json:"skipped"`
			ConflictSkipped int `json:"conflict_skipped"`
			Archived        int `json:"archived"`
			Errors          int `json:"errors"`
		} `json:"counts"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("stdout is not JSON: %v\nstdout=%s", err, stdout)
	}
	if doc.Counts.Migrated < 1 {
		t.Fatalf("expected migrated >= 1, got %+v\nstdout=%s", doc.Counts, stdout)
	}
	if strings.Contains(stdout, "tg-secret") || strings.Contains(stdout, "sk-deadbeef") {
		t.Fatalf("stdout leaked secret bytes: %s", stdout)
	}

	// Destination should now contain a config.toml under <dest>/config.toml.
	if _, statErr := os.Stat(filepath.Join(dest, "config.toml")); statErr != nil {
		t.Fatalf("dest/config.toml missing after --yes: %v", statErr)
	}
}

func TestMigrateHermesWriter_UsesHermesHomeEnvSource(t *testing.T) {
	root := setupMigrateHermesWriterEnv(t)
	src := filepath.Join(root, "hermes-env-src")
	writeMigrateWriterFixture(t, src)
	t.Setenv("HERMES_HOME", src)
	dest := filepath.Join(root, "dest")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}

	_, stdout, stderr, err := executeMigrateHermes("hermes", "--yes", "--dest", dest)
	if err != nil {
		t.Fatalf("migrate hermes --yes using HERMES_HOME: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if _, statErr := os.Stat(filepath.Join(dest, "config.toml")); statErr != nil {
		t.Fatalf("dest/config.toml missing after HERMES_HOME apply: %v", statErr)
	}
	if strings.Contains(stdout, "tg-secret") || strings.Contains(stdout, "sk-deadbeef") {
		t.Fatalf("stdout leaked secret bytes: %s", stdout)
	}
}

func TestMigrateHermesWriter_UsesDefaultHomeDotHermesSource(t *testing.T) {
	root := setupMigrateHermesWriterEnv(t)
	src := filepath.Join(os.Getenv("HOME"), ".hermes")
	writeMigrateWriterFixture(t, src)
	dest := filepath.Join(root, "dest")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}

	_, stdout, stderr, err := executeMigrateHermes("hermes", "--yes", "--dest", dest)
	if err != nil {
		t.Fatalf("migrate hermes --yes using ~/.hermes: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if _, statErr := os.Stat(filepath.Join(dest, "config.toml")); statErr != nil {
		t.Fatalf("dest/config.toml missing after ~/.hermes apply: %v", statErr)
	}
	if strings.Contains(stdout, "tg-secret") || strings.Contains(stdout, "sk-deadbeef") {
		t.Fatalf("stdout leaked secret bytes: %s", stdout)
	}
}

func TestMigrateHermesWriter_NoDiscoveredSourceGivesHelpfulError(t *testing.T) {
	root := setupMigrateHermesWriterEnv(t)
	dest := filepath.Join(root, "dest")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}

	_, stdout, stderr, err := executeMigrateHermes("hermes", "--yes", "--dest", dest)
	if err == nil {
		t.Fatalf("missing source should fail; stdout=%s stderr=%s", stdout, stderr)
	}
	combined := err.Error() + stderr
	if !strings.Contains(combined, "no Hermes source found") || !strings.Contains(combined, "--source /path/to/hermes-home") || !strings.Contains(combined, "HERMES_HOME") {
		t.Fatalf("error should explain how to configure source; got: %s", combined)
	}
	if strings.Contains(combined, "--source is required") {
		t.Fatalf("error regressed to hard-required --source wording: %s", combined)
	}
}

func TestMigrateHermesWriter_RequiresYesGuidance(t *testing.T) {
	root := setupMigrateHermesWriterEnv(t)
	src := filepath.Join(root, "hermes-src")
	writeMigrateWriterFixture(t, src)
	dest := filepath.Join(root, "dest")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatalf("mkdir dest: %v", err)
	}

	_, stdout, stderr, err := executeMigrateHermes("hermes", "--source", src, "--dest", dest)
	if err == nil {
		t.Fatalf("missing --yes/--dry-run should fail; stdout=%s stderr=%s", stdout, stderr)
	}
	combined := err.Error() + stderr
	if !strings.Contains(combined, "--yes") || !strings.Contains(combined, "--dry-run") {
		t.Fatalf("error must reference --yes and --dry-run; got: %s", combined)
	}
}
