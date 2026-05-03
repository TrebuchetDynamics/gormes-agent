package hermes

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeAPIKey is a synthetic secret used in writer fixtures to verify
// secret values never appear in the outcome JSON. Real Hermes installs
// would have similar shapes (sk-..., ANTHROPIC_API_KEY=...).
const fakeAPIKey = "sk-deadbeef-do-not-leak"

// writerSampleConfig produces a Hermes config.yaml with one importable
// section, one archived key, and one unknown section.
func writerSampleConfig() string {
	return `model: gpt-4.1-mini
gateway:
  notify_interval: 180
_config_version: 26
unknown_section:
  ignored: true
`
}

// writerSampleEnv contains importable keys (TELEGRAM_TOKEN, *_API_KEY),
// an unsupported key (RANDOM_USER_VAR), and one whose target may
// already exist on the destination (DISCORD_TOKEN -> GORMES_DISCORD_TOKEN).
func writerSampleEnv() string {
	return `TELEGRAM_TOKEN=tg-secret-do-not-leak
DISCORD_TOKEN=dc-secret-do-not-leak
ANTHROPIC_API_KEY=` + fakeAPIKey + `
RANDOM_USER_VAR=plainvalue
`
}

// buildWriterFixtureManifest assembles a manifest + raw source bytes
// pair from a temp source dir so each test can exercise ApplyManifest
// without re-implementing source discovery.
func buildWriterFixtureManifest(t *testing.T, existing map[string]string) (Manifest, map[string][]byte, string) {
	t.Helper()
	root := t.TempDir()
	src := filepath.Join(root, "hermes-src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	cfgBody := []byte(writerSampleConfig())
	envBody := []byte(writerSampleEnv())
	if err := os.WriteFile(filepath.Join(src, "config.yaml"), cfgBody, 0o600); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, ".env"), envBody, 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	t.Setenv("HERMES_HOME", "")

	m, err := BuildManifest(Options{Source: src, ExistingGormesEnv: existing})
	if err != nil {
		t.Fatalf("BuildManifest: %v", err)
	}
	src2 := map[string][]byte{
		"config.yaml": cfgBody,
		".env":        envBody,
	}
	return *m, src2, root
}

func writerDestPaths(t *testing.T, root string) (cfgDir, envFile string) {
	t.Helper()
	cfgDir = filepath.Join(root, "dest-config")
	envFile = filepath.Join(root, "dest-config", ".env")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir dest-config: %v", err)
	}
	return cfgDir, envFile
}

func TestHermesConfigWriter_RequiresYes(t *testing.T) {
	manifest, src, root := buildWriterFixtureManifest(t, nil)
	cfgDir, envFile := writerDestPaths(t, root)

	out, err := ApplyManifest(WriteRequest{
		Manifest:          manifest,
		DestConfigDir:     cfgDir,
		DestEnvFile:       envFile,
		SourceConfigBytes: src,
		// Yes intentionally false
	})
	if err == nil {
		t.Fatalf("ApplyManifest without --yes must return guidance error; got nil and outcome=%+v", out)
	}
	msg := err.Error()
	if !strings.Contains(msg, "--yes") || !strings.Contains(msg, "--dry-run") {
		t.Fatalf("error must mention --yes and --dry-run guidance, got: %v", err)
	}
	// No file should have been written under cfgDir besides the dir itself.
	entries, _ := os.ReadDir(cfgDir)
	if len(entries) != 0 {
		t.Fatalf("dest config dir must be untouched without --yes, got entries=%v", entries)
	}
	if _, statErr := os.Stat(envFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("dest env file must not exist without --yes, stat err=%v", statErr)
	}
}

func TestHermesConfigWriter_AppliesImportableConfig(t *testing.T) {
	manifest, src, root := buildWriterFixtureManifest(t, nil)
	cfgDir, envFile := writerDestPaths(t, root)

	out, err := ApplyManifest(WriteRequest{
		Manifest:          manifest,
		DestConfigDir:     cfgDir,
		DestEnvFile:       envFile,
		SourceConfigBytes: src,
		Yes:               true,
	})
	if err != nil {
		t.Fatalf("ApplyManifest: %v", err)
	}

	// config.toml created with importable sections from the manifest.
	cfgPath := filepath.Join(cfgDir, "config.toml")
	body, readErr := os.ReadFile(cfgPath)
	if readErr != nil {
		t.Fatalf("read dest config.toml: %v", readErr)
	}
	if !strings.Contains(string(body), "[hermes]") && !strings.Contains(string(body), "model") {
		t.Fatalf("config.toml missing imported model section: %s", body)
	}
	if !strings.Contains(string(body), "[gateway]") {
		t.Fatalf("config.toml missing imported gateway section: %s", body)
	}

	// dest .env contains importable keys, secret bytes preserved on disk.
	envBody, envErr := os.ReadFile(envFile)
	if envErr != nil {
		t.Fatalf("read dest .env: %v", envErr)
	}
	if !strings.Contains(string(envBody), "GORMES_TELEGRAM_TOKEN=tg-secret-do-not-leak") {
		t.Fatalf("dest .env missing GORMES_TELEGRAM_TOKEN value: %s", envBody)
	}
	if !strings.Contains(string(envBody), "ANTHROPIC_API_KEY="+fakeAPIKey) {
		t.Fatalf("dest .env missing ANTHROPIC_API_KEY value: %s", envBody)
	}
	if strings.Contains(string(envBody), "RANDOM_USER_VAR=") {
		t.Fatalf("dest .env must not include unsupported key RANDOM_USER_VAR: %s", envBody)
	}

	// Outcome counts: at least one config and one env migrated.
	if out.Counts.Migrated < 2 {
		t.Fatalf("expected migrated >= 2 (config + env), got %+v", out.Counts)
	}
	if got, ok := out.ConfigWritten["model"]; !ok || got != "migrated" {
		t.Fatalf("ConfigWritten[model] = %q ok=%v, want migrated/true", got, ok)
	}
	if got, ok := out.EnvWritten["GORMES_TELEGRAM_TOKEN"]; !ok || got != "migrated" {
		t.Fatalf("EnvWritten[GORMES_TELEGRAM_TOKEN] = %q ok=%v, want migrated/true", got, ok)
	}
}

func TestHermesConfigWriter_MapsModelObjectAndDisplayToNativeConfig(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, "hermes-src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	cfgBody := []byte(`model:
  default: gpt-5.5
  provider: openai-codex
display:
  tool_progress: new
  tool_progress_command: true
  platforms:
    telegram:
      tool_progress: false
  tool_progress_overrides:
    discord: verbose
`)
	if err := os.WriteFile(filepath.Join(srcDir, "config.yaml"), cfgBody, 0o600); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}
	t.Setenv("HERMES_HOME", "")
	manifest, err := BuildManifest(Options{Source: srcDir})
	if err != nil {
		t.Fatalf("BuildManifest: %v", err)
	}
	cfgDir, envFile := writerDestPaths(t, root)

	_, err = ApplyManifest(WriteRequest{
		Manifest:          *manifest,
		DestConfigDir:     cfgDir,
		DestEnvFile:       envFile,
		SourceConfigBytes: map[string][]byte{"config.yaml": cfgBody},
		Yes:               true,
	})
	if err != nil {
		t.Fatalf("ApplyManifest: %v", err)
	}

	body, err := os.ReadFile(filepath.Join(cfgDir, "config.toml"))
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	got := string(body)
	for _, want := range []string{
		"[hermes]",
		"model = 'gpt-5.5'",
		"provider = 'openai-codex'",
		"[display]",
		"tool_progress = 'new'",
		"tool_progress_command = true",
		"[display.platforms.discord]",
		"tool_progress = 'verbose'",
		"[display.platforms.telegram]",
		"tool_progress = 'off'",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("config.toml missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "[tui]") || strings.Contains(got, "tool_progress_overrides") {
		t.Fatalf("config.toml kept legacy display destination/shape:\n%s", got)
	}
}

func TestHermesConfigWriter_BackupBeforeOverwrite(t *testing.T) {
	manifest, src, root := buildWriterFixtureManifest(t, nil)
	cfgDir, envFile := writerDestPaths(t, root)

	// Pre-existing destination config.toml with a hermes-section value
	// the writer would touch.
	cfgPath := filepath.Join(cfgDir, "config.toml")
	preExisting := []byte("[hermes]\nmodel = \"pre-existing-model\"\n")
	if err := os.WriteFile(cfgPath, preExisting, 0o600); err != nil {
		t.Fatalf("seed dest config.toml: %v", err)
	}

	out, err := ApplyManifest(WriteRequest{
		Manifest:          manifest,
		DestConfigDir:     cfgDir,
		DestEnvFile:       envFile,
		SourceConfigBytes: src,
		Yes:               true,
		Overwrite:         true,
	})
	if err != nil {
		t.Fatalf("ApplyManifest: %v", err)
	}

	// At least one backup must have been created and reference cfgPath.
	if len(out.Backups) == 0 {
		t.Fatalf("expected at least one backup file in outcome, got none; outcome=%+v", out)
	}
	var found string
	for _, b := range out.Backups {
		if strings.HasPrefix(b, cfgPath) || strings.Contains(b, "config.toml") {
			found = b
			break
		}
	}
	if found == "" {
		t.Fatalf("no backup references config.toml: %v", out.Backups)
	}
	got, err := os.ReadFile(found)
	if err != nil {
		t.Fatalf("read backup %q: %v", found, err)
	}
	if string(got) != string(preExisting) {
		t.Fatalf("backup contents differ from pre-existing file:\nwant=%q\ngot=%q", preExisting, got)
	}
	// Counts surface archived count for the backup.
	if out.Counts.Archived < 1 {
		t.Fatalf("Counts.Archived must be >= 1 when a backup is created, got %+v", out.Counts)
	}
}

func TestHermesConfigWriter_ExistingEnvConflictWithoutOverwrite(t *testing.T) {
	existing := map[string]string{
		"GORMES_TELEGRAM_TOKEN": "preset-tg-value",
	}
	manifest, src, root := buildWriterFixtureManifest(t, existing)
	cfgDir, envFile := writerDestPaths(t, root)

	out, err := ApplyManifest(WriteRequest{
		Manifest:          manifest,
		DestConfigDir:     cfgDir,
		DestEnvFile:       envFile,
		ExistingGormesEnv: existing,
		SourceConfigBytes: src,
		Yes:               true,
		// Overwrite intentionally false
	})
	if err != nil {
		t.Fatalf("ApplyManifest: %v", err)
	}
	got := out.EnvWritten["GORMES_TELEGRAM_TOKEN"]
	if got != "conflict_skipped" {
		t.Fatalf("GORMES_TELEGRAM_TOKEN disposition = %q, want conflict_skipped; outcome=%+v", got, out)
	}
	if out.Counts.ConflictSkipped < 1 {
		t.Fatalf("Counts.ConflictSkipped = %d, want >=1; outcome=%+v", out.Counts.ConflictSkipped, out)
	}

	// Now retry with --overwrite. We need a fresh manifest because the
	// dry-run manifest still classifies this key as "conflict"; the
	// writer must honor --overwrite to flip conflicts to migrated.
	manifest2, src2, root2 := buildWriterFixtureManifest(t, existing)
	cfgDir2, envFile2 := writerDestPaths(t, root2)
	out2, err := ApplyManifest(WriteRequest{
		Manifest:          manifest2,
		DestConfigDir:     cfgDir2,
		DestEnvFile:       envFile2,
		ExistingGormesEnv: existing,
		SourceConfigBytes: src2,
		Yes:               true,
		Overwrite:         true,
	})
	if err != nil {
		t.Fatalf("ApplyManifest with overwrite: %v", err)
	}
	if got := out2.EnvWritten["GORMES_TELEGRAM_TOKEN"]; got != "migrated" {
		t.Fatalf("with --overwrite: GORMES_TELEGRAM_TOKEN disposition = %q, want migrated; outcome=%+v", got, out2)
	}
}

func TestHermesConfigWriter_OutcomeJSONHasNoSecrets(t *testing.T) {
	manifest, src, root := buildWriterFixtureManifest(t, nil)
	cfgDir, envFile := writerDestPaths(t, root)

	out, err := ApplyManifest(WriteRequest{
		Manifest:          manifest,
		DestConfigDir:     cfgDir,
		DestEnvFile:       envFile,
		SourceConfigBytes: src,
		Yes:               true,
	})
	if err != nil {
		t.Fatalf("ApplyManifest: %v", err)
	}

	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal outcome: %v", err)
	}
	body := string(raw)
	for _, leak := range []string{
		fakeAPIKey,
		"tg-secret-do-not-leak",
		"dc-secret-do-not-leak",
	} {
		if strings.Contains(body, leak) {
			t.Fatalf("outcome JSON leaked secret %q in body: %s", leak, body)
		}
	}
}
