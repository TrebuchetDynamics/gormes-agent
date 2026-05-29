package hermes

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// writeFile is a test helper that writes content under tempdir-rooted paths
// only. The migrate dry-run manifest never reads ~/.hermes from tests.
func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func sampleConfigYAML() string {
	return `model: gpt-4.1-mini
providers:
  openrouter:
    api_key: sk-or-secret-redacted-in-env
custom_provider:
  base_url: https://example.invalid/v1
  api_key_env: CUSTOM_KEY
terminal:
  backend: local
  timeout: 180
display:
  theme: dark
gateway:
  notify_interval: 180
memory:
  enabled: true
unknown_top_level_section:
  ignored: true
`
}

func sampleEnv() string {
	return `# Hermes .env fixture
TELEGRAM_TOKEN=123:abc-secret
TELEGRAM_BOT_TOKEN=123:bot-secret
TELEGRAM_HOME_CHANNEL=123456789
TELEGRAM_ALLOWED_USERS=123456789,987654321
DISCORD_TOKEN=disc-secret
OPENROUTER_API_KEY=sk-or-real-secret
RANDOM_USER_VAR=plainvalue
`
}

func TestHermesMigrationManifest_SourceDiscoveryOrderRecordsAllPaths(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "explicit-src")
	hermesHome := filepath.Join(root, "hermes-home")
	homeDotHermes := filepath.Join(root, "fake-home", ".hermes")
	writeFile(t, filepath.Join(src, "config.yaml"), sampleConfigYAML())
	writeFile(t, filepath.Join(hermesHome, "config.yaml"), sampleConfigYAML())
	writeFile(t, filepath.Join(homeDotHermes, "config.yaml"), sampleConfigYAML())

	t.Setenv("HERMES_HOME", hermesHome)
	t.Setenv("HOME", filepath.Dir(homeDotHermes))

	m, err := BuildManifest(Options{Source: src})
	if err != nil {
		t.Fatalf("BuildManifest with --source: %v", err)
	}

	if m.Source.Selected != "explicit_source" {
		t.Fatalf("Selected source = %q, want explicit_source", m.Source.Selected)
	}
	if m.Source.SelectedPath != src {
		t.Fatalf("SelectedPath = %q, want %q", m.Source.SelectedPath, src)
	}
	if got := candidatePath(m, "explicit_source"); got != src {
		t.Fatalf("explicit_source candidate = %q, want %q", got, src)
	}
	if got := candidatePath(m, "hermes_home_env"); got != hermesHome {
		t.Fatalf("hermes_home_env candidate = %q, want %q", got, hermesHome)
	}
	if got := candidatePath(m, "user_home_dot_hermes"); got != homeDotHermes {
		t.Fatalf("user_home_dot_hermes candidate = %q, want %q", got, homeDotHermes)
	}
	for _, c := range m.Source.Candidates {
		if !c.Found {
			t.Fatalf("candidate %s not marked Found despite existing fixture: %+v", c.Origin, c)
		}
	}
}

func TestHermesMigrationManifest_FallsBackToHermesHomeEnvWhenNoSource(t *testing.T) {
	root := t.TempDir()
	hermesHome := filepath.Join(root, "hermes-home")
	homeDotHermes := filepath.Join(root, "fake-home", ".hermes")
	writeFile(t, filepath.Join(hermesHome, "config.yaml"), sampleConfigYAML())
	writeFile(t, filepath.Join(homeDotHermes, "config.yaml"), sampleConfigYAML())

	t.Setenv("HERMES_HOME", hermesHome)
	t.Setenv("HOME", filepath.Dir(homeDotHermes))

	m, err := BuildManifest(Options{})
	if err != nil {
		t.Fatalf("BuildManifest: %v", err)
	}
	if m.Source.Selected != "hermes_home_env" {
		t.Fatalf("Selected = %q, want hermes_home_env", m.Source.Selected)
	}
	if m.Source.SelectedPath != hermesHome {
		t.Fatalf("SelectedPath = %q, want %q", m.Source.SelectedPath, hermesHome)
	}
}

func TestHermesMigrationManifest_FallsBackToUserDotHermesWhenNoEnv(t *testing.T) {
	root := t.TempDir()
	homeDotHermes := filepath.Join(root, "fake-home", ".hermes")
	writeFile(t, filepath.Join(homeDotHermes, "config.yaml"), sampleConfigYAML())

	t.Setenv("HERMES_HOME", "")
	t.Setenv("HOME", filepath.Dir(homeDotHermes))

	m, err := BuildManifest(Options{})
	if err != nil {
		t.Fatalf("BuildManifest: %v", err)
	}
	if m.Source.Selected != "user_home_dot_hermes" {
		t.Fatalf("Selected = %q, want user_home_dot_hermes", m.Source.Selected)
	}
	if m.Source.SelectedPath != homeDotHermes {
		t.Fatalf("SelectedPath = %q, want %q", m.Source.SelectedPath, homeDotHermes)
	}
}

func TestHermesMigrationManifest_ConfigYAMLClassifiesSupportedKeys(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	writeFile(t, filepath.Join(src, "config.yaml"), sampleConfigYAML())
	t.Setenv("HERMES_HOME", "")

	m, err := BuildManifest(Options{Source: src})
	if err != nil {
		t.Fatalf("BuildManifest: %v", err)
	}

	wantImportable := map[string]string{
		"model":           "hermes.model",
		"providers":       "hermes.providers",
		"custom_provider": "hermes.custom_provider",
		"terminal":        "terminal",
		"display":         "display",
		"gateway":         "gateway",
		"memory":          "memory",
	}
	for hermesKey, gormesPath := range wantImportable {
		entry := findConfigEntry(m, hermesKey)
		if entry == nil {
			t.Fatalf("missing config entry for hermes key %q", hermesKey)
		}
		if entry.Disposition != "importable" {
			t.Fatalf("hermes key %q disposition = %q, want importable", hermesKey, entry.Disposition)
		}
		if entry.GormesPath != gormesPath {
			t.Fatalf("hermes key %q gormes_path = %q, want %q", hermesKey, entry.GormesPath, gormesPath)
		}
	}

	unknown := findConfigEntry(m, "unknown_top_level_section")
	if unknown == nil {
		t.Fatalf("missing entry for unknown_top_level_section")
	}
	if unknown.Disposition != "skipped" {
		t.Fatalf("unknown disposition = %q, want skipped", unknown.Disposition)
	}
	if !strings.Contains(strings.ToLower(unknown.Reason), "not in supported migration set") {
		t.Fatalf("unknown reason missing supported-set evidence: %q", unknown.Reason)
	}
}

func TestHermesMigrationManifest_ModelProviderAndDisplayNativeTargets(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	writeFile(t, filepath.Join(src, "config.yaml"), `
model:
  default: gpt-5.5
  provider: openai-codex
display:
  tool_progress: new
  tool_progress_command: true
`)
	t.Setenv("HERMES_HOME", "")

	m, err := BuildManifest(Options{Source: src})
	if err != nil {
		t.Fatalf("BuildManifest: %v", err)
	}
	model := findConfigEntry(m, "model")
	if model == nil {
		t.Fatalf("missing model entry")
	}
	if model.Disposition != "importable" {
		t.Fatalf("model disposition = %q, want importable", model.Disposition)
	}
	if model.GormesPath != "hermes.model,llm.provider" {
		t.Fatalf("model gormes_path = %q, want hermes.model,llm.provider", model.GormesPath)
	}
	display := findConfigEntry(m, "display")
	if display == nil {
		t.Fatalf("missing display entry")
	}
	if display.GormesPath != "display" {
		t.Fatalf("display gormes_path = %q, want display", display.GormesPath)
	}
}

func TestHermesMigrationManifest_ConfigYAMLArchivesNotYetMigrated(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	body := sampleConfigYAML() + "\n_config_version: 26\n"
	writeFile(t, filepath.Join(src, "config.yaml"), body)
	t.Setenv("HERMES_HOME", "")

	m, err := BuildManifest(Options{Source: src})
	if err != nil {
		t.Fatalf("BuildManifest: %v", err)
	}
	entry := findConfigEntry(m, "_config_version")
	if entry == nil {
		t.Fatalf("missing _config_version entry")
	}
	if entry.Disposition != "archived" {
		t.Fatalf("_config_version disposition = %q, want archived", entry.Disposition)
	}
}

func TestHermesMigrationManifest_DotenvKeysClassifiedAndRedacted(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	writeFile(t, filepath.Join(src, ".env"), sampleEnv())
	t.Setenv("HERMES_HOME", "")

	m, err := BuildManifest(Options{Source: src})
	if err != nil {
		t.Fatalf("BuildManifest: %v", err)
	}
	wantTargets := map[string]string{
		"TELEGRAM_TOKEN":     "GORMES_TELEGRAM_BOT_TOKEN",
		"TELEGRAM_BOT_TOKEN": "GORMES_TELEGRAM_BOT_TOKEN",
		"DISCORD_TOKEN":      "GORMES_DISCORD_TOKEN",
		"OPENROUTER_API_KEY": "OPENROUTER_API_KEY",
	}
	for srcKey, gormesEnv := range wantTargets {
		entry := findEnvEntry(m, srcKey)
		if entry == nil {
			t.Fatalf("missing env entry for %s", srcKey)
		}
		if entry.Disposition != "importable" {
			t.Fatalf("%s disposition = %q, want importable", srcKey, entry.Disposition)
		}
		if entry.GormesEnv != gormesEnv {
			t.Fatalf("%s gormes_env = %q, want %q", srcKey, entry.GormesEnv, gormesEnv)
		}
		if entry.RedactedValue != "[REDACTED]" {
			t.Fatalf("%s redacted_value = %q, want [REDACTED]", srcKey, entry.RedactedValue)
		}
		// Raw secret bytes must never appear in the manifest.
		raw, _ := json.Marshal(entry)
		if strings.Contains(string(raw), "secret") || strings.Contains(string(raw), "abc") {
			t.Fatalf("env entry leaked raw secret material: %s", raw)
		}
	}
	unknownEnv := findEnvEntry(m, "RANDOM_USER_VAR")
	if unknownEnv == nil || unknownEnv.Disposition != "skipped" {
		t.Fatalf("RANDOM_USER_VAR should be skipped, got %+v", unknownEnv)
	}
}

func TestHermesTelegramEnvMappingTargetsStructuredConfig(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	writeFile(t, filepath.Join(src, ".env"), `
TELEGRAM_HOME_CHANNEL=-1001234567890
TELEGRAM_HOME_CHANNEL_NAME=alerts
TELEGRAM_HOME_CHANNEL_THREAD_ID=42
TELEGRAM_ALLOWED_USERS=6586915095,12345
`)
	t.Setenv("HERMES_HOME", "")

	m, err := BuildManifest(Options{Source: src})
	if err != nil {
		t.Fatalf("BuildManifest: %v", err)
	}
	wantPaths := map[string]string{
		"TELEGRAM_HOME_CHANNEL":           "telegram.home_channel.chat_id",
		"TELEGRAM_HOME_CHANNEL_NAME":      "telegram.home_channel.name",
		"TELEGRAM_HOME_CHANNEL_THREAD_ID": "telegram.home_channel.thread_id",
		"TELEGRAM_ALLOWED_USERS":          "telegram.allowed_user_ids",
	}
	for srcKey, gormesPath := range wantPaths {
		entry := findEnvEntry(m, srcKey)
		if entry == nil {
			t.Fatalf("missing env entry for %s", srcKey)
		}
		if entry.Disposition != "importable" {
			t.Fatalf("%s disposition = %q, want importable", srcKey, entry.Disposition)
		}
		if entry.GormesPath != gormesPath {
			t.Fatalf("%s gormes_path = %q, want %q", srcKey, entry.GormesPath, gormesPath)
		}
		if entry.GormesEnv != "" {
			t.Fatalf("%s gormes_env = %q, want TOML-only mapping", srcKey, entry.GormesEnv)
		}
	}
}

func TestHermesMigrationManifest_DotenvConflictAgainstExistingTarget(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	writeFile(t, filepath.Join(src, ".env"), "TELEGRAM_TOKEN=fixture-secret\n")
	t.Setenv("HERMES_HOME", "")

	m, err := BuildManifest(Options{
		Source: src,
		ExistingGormesEnv: map[string]string{
			"GORMES_TELEGRAM_BOT_TOKEN": "already-set-something",
		},
	})
	if err != nil {
		t.Fatalf("BuildManifest: %v", err)
	}
	entry := findEnvEntry(m, "TELEGRAM_TOKEN")
	if entry == nil || entry.Disposition != "conflict" {
		t.Fatalf("TELEGRAM_TOKEN should be conflict, got %+v", entry)
	}
	if entry.GormesEnv != "GORMES_TELEGRAM_BOT_TOKEN" {
		t.Fatalf("conflict entry missing GormesEnv mapping, got %+v", entry)
	}
	if !entry.ConflictWithExisting {
		t.Fatalf("ConflictWithExisting must be true on conflict, got %+v", entry)
	}
	if entry.RedactedValue != "[REDACTED]" {
		t.Fatalf("conflict redacted_value = %q, want [REDACTED]", entry.RedactedValue)
	}
}

func TestHermesMigrationManifest_CountsMatchEntries(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	writeFile(t, filepath.Join(src, "config.yaml"), sampleConfigYAML())
	writeFile(t, filepath.Join(src, ".env"), sampleEnv())
	t.Setenv("HERMES_HOME", "")

	m, err := BuildManifest(Options{
		Source: src,
		ExistingGormesEnv: map[string]string{
			"GORMES_DISCORD_TOKEN": "preset",
		},
	})
	if err != nil {
		t.Fatalf("BuildManifest: %v", err)
	}
	c := m.Counts
	migrated, skipped, conflict, errCount := 0, 0, 0, 0
	for _, e := range m.Config {
		switch e.Disposition {
		case "importable":
			migrated++
		case "skipped", "archived":
			skipped++
		case "conflict":
			conflict++
		case "error":
			errCount++
		}
	}
	for _, e := range m.Env {
		switch e.Disposition {
		case "importable":
			migrated++
		case "skipped", "archived":
			skipped++
		case "conflict":
			conflict++
		case "error":
			errCount++
		}
	}
	if c.Migrated != migrated || c.Skipped != skipped || c.Conflict != conflict || c.Errors != errCount {
		t.Fatalf("Counts mismatch: counts=%+v derived migrated=%d skipped=%d conflict=%d err=%d",
			c, migrated, skipped, conflict, errCount)
	}
	if c.Conflict < 1 {
		t.Fatalf("expected at least 1 conflict (DISCORD_TOKEN preset), got %+v", c)
	}
}

func TestHermesMigrationManifest_InvalidYAMLProducesEntryNotPanic(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	writeFile(t, filepath.Join(src, "config.yaml"), "this: is: not: valid: yaml:\n  - [")
	t.Setenv("HERMES_HOME", "")

	m, err := BuildManifest(Options{Source: src})
	if err != nil {
		t.Fatalf("invalid yaml should not error the manifest builder, got: %v", err)
	}
	if len(m.Errors) == 0 {
		t.Fatalf("expected at least one error entry for invalid yaml, got none")
	}
	found := false
	for _, e := range m.Errors {
		if e.Source == "config.yaml" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("config.yaml error not recorded in manifest.Errors: %+v", m.Errors)
	}
}

func TestHermesMigrationManifest_MissingSourceErrorsExplicit(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "does-not-exist")
	t.Setenv("HERMES_HOME", "")

	_, err := BuildManifest(Options{Source: missing})
	if err == nil {
		t.Fatalf("BuildManifest with missing --source should error")
	}
	if !strings.Contains(err.Error(), missing) {
		t.Fatalf("error should reference path %q, got %v", missing, err)
	}
}

func TestHermesMigrationManifest_NoCandidatesYieldsErrorlessEmptyManifest(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HERMES_HOME", "")
	t.Setenv("HOME", filepath.Join(root, "no-such"))

	m, err := BuildManifest(Options{})
	if err != nil {
		t.Fatalf("no candidates should not error, got: %v", err)
	}
	if m.Source.Selected != "" {
		t.Fatalf("Selected should be empty when no candidates resolve, got %q", m.Source.Selected)
	}
	if len(m.Config) != 0 || len(m.Env) != 0 {
		t.Fatalf("expected empty manifest, got config=%d env=%d", len(m.Config), len(m.Env))
	}
}

func TestHermesMigrationManifest_DryRunWritesNothingUnderXDGRoots(t *testing.T) {
	root := t.TempDir()
	cfg := filepath.Join(root, "xdg-config")
	data := filepath.Join(root, "xdg-data")
	state := filepath.Join(root, "xdg-state")
	src := filepath.Join(root, "src")
	for _, d := range []string{cfg, data, state} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	writeFile(t, filepath.Join(src, "config.yaml"), sampleConfigYAML())
	writeFile(t, filepath.Join(src, ".env"), sampleEnv())
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("XDG_STATE_HOME", state)
	t.Setenv("HERMES_HOME", "")

	beforeCfg := snapshotDir(t, cfg)
	beforeData := snapshotDir(t, data)
	beforeState := snapshotDir(t, state)

	if _, err := BuildManifest(Options{Source: src}); err != nil {
		t.Fatalf("BuildManifest: %v", err)
	}

	if got := snapshotDir(t, cfg); !equalSnapshots(got, beforeCfg) {
		t.Fatalf("XDG_CONFIG_HOME mutated during dry-run: before=%v after=%v", beforeCfg, got)
	}
	if got := snapshotDir(t, data); !equalSnapshots(got, beforeData) {
		t.Fatalf("XDG_DATA_HOME mutated during dry-run: before=%v after=%v", beforeData, got)
	}
	if got := snapshotDir(t, state); !equalSnapshots(got, beforeState) {
		t.Fatalf("XDG_STATE_HOME mutated during dry-run: before=%v after=%v", beforeState, got)
	}
}

// helpers

func candidatePath(m *Manifest, origin string) string {
	for _, c := range m.Source.Candidates {
		if c.Origin == origin {
			return c.Path
		}
	}
	return ""
}

func findConfigEntry(m *Manifest, key string) *ConfigEntry {
	for i := range m.Config {
		if m.Config[i].HermesKey == key {
			return &m.Config[i]
		}
	}
	return nil
}

func findEnvEntry(m *Manifest, key string) *EnvEntry {
	for i := range m.Env {
		if m.Env[i].HermesKey == key {
			return &m.Env[i]
		}
	}
	return nil
}

func snapshotDir(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	if err := filepath.Walk(root, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		out = append(out, path)
		return nil
	}); err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	sort.Strings(out)
	return out
}

func equalSnapshots(a, b []string) bool {
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
