package openclaw

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const fakeOpenClawAPIKey = "sk-openclaw-do-not-leak"

func writeApplyFixture(t *testing.T, src string, existing map[string]string) (Manifest, map[string][]byte) {
	t.Helper()
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	cfgBody := []byte(`model: gpt-4.1-mini
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
`)
	envBody := []byte("TELEGRAM_BOT_TOKEN=plain-telegram-token\n" +
		"DISCORD_BOT_TOKEN=plain-discord-token\n" +
		"OPENROUTER_API_KEY=" + fakeOpenClawAPIKey + "\n" +
		"RANDOM_USER_VAR=plainvalue\n")
	if err := os.WriteFile(filepath.Join(src, "config.yaml"), cfgBody, 0o600); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, ".env"), envBody, 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "MEMORY.md"), []byte("# memory\nhello\n"), 0o600); err != nil {
		t.Fatalf("write MEMORY.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "USER.md"), []byte("# user\nme\n"), 0o600); err != nil {
		t.Fatalf("write USER.md: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(src, "skills", "demo"), 0o755); err != nil {
		t.Fatalf("mkdir skills: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "skills", "demo", "SKILL.md"), []byte("skill body\n"), 0o600); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	t.Setenv("HOME", filepath.Join(t.TempDir(), "fake-home"))
	m, err := BuildManifest(Options{Source: src, ExistingGormesEnv: existing})
	if err != nil {
		t.Fatalf("BuildManifest: %v", err)
	}
	return *m, map[string][]byte{
		"config.yaml": cfgBody,
		".env":        envBody,
	}
}

func applyDestPaths(t *testing.T, root string) (cfgDir, envFile, skillsDir, memoryDir, reportRoot string) {
	t.Helper()
	cfgDir = filepath.Join(root, "dest-config")
	envFile = filepath.Join(cfgDir, ".env")
	skillsDir = filepath.Join(root, "dest-skills")
	memoryDir = filepath.Join(root, "dest-memory")
	reportRoot = filepath.Join(root, "state", "gormes", "migrations", "openclaw")
	for _, d := range []string{cfgDir, skillsDir, memoryDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	return
}

func fixedNow(t *testing.T) func() time.Time {
	t.Helper()
	stamp := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	return func() time.Time { return stamp }
}

func sourceRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "openclaw-src")
}

func TestOpenClawMigrationApply_RequiresYes(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "openclaw-src")
	manifest, srcBytes := writeApplyFixture(t, src, nil)
	cfgDir, envFile, skillsDir, memoryDir, reportRoot := applyDestPaths(t, root)

	out, err := ApplyManifest(ApplyRequest{
		Manifest:          manifest,
		DestConfigDir:     cfgDir,
		DestEnvFile:       envFile,
		DestSkillsDir:     skillsDir,
		DestMemoryDir:     memoryDir,
		ReportRootDir:     reportRoot,
		SourceConfigBytes: srcBytes,
		SourceRoot:        src,
		SecretsEnabled:    true,
		Now:               fixedNow(t),
		// Yes intentionally false
	})
	if err == nil {
		t.Fatalf("ApplyManifest without --yes must return error; outcome=%+v", out)
	}
	msg := err.Error()
	if !strings.Contains(msg, "--yes") {
		t.Fatalf("error must mention --yes; got: %v", err)
	}
	// Nothing should have been written to dest.
	for _, dir := range []string{cfgDir, skillsDir, memoryDir, reportRoot} {
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			t.Fatalf("dest %s must be untouched without --yes, found entry=%s", dir, e.Name())
		}
	}
	if _, statErr := os.Stat(envFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("dest env file must not exist without --yes, stat err=%v", statErr)
	}
}

func TestOpenClawMigrationApply_AppliesAllFiveSurfaces(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "openclaw-src")
	manifest, srcBytes := writeApplyFixture(t, src, nil)
	cfgDir, envFile, skillsDir, memoryDir, reportRoot := applyDestPaths(t, root)

	out, err := ApplyManifest(ApplyRequest{
		Manifest:          manifest,
		DestConfigDir:     cfgDir,
		DestEnvFile:       envFile,
		DestSkillsDir:     skillsDir,
		DestMemoryDir:     memoryDir,
		ReportRootDir:     reportRoot,
		SourceConfigBytes: srcBytes,
		SourceRoot:        src,
		SecretsEnabled:    true,
		Yes:               true,
		Now:               fixedNow(t),
	})
	if err != nil {
		t.Fatalf("ApplyManifest: %v", err)
	}

	// Config: a config.toml lands in DestConfigDir.
	cfgPath := filepath.Join(cfgDir, "config.toml")
	if _, statErr := os.Stat(cfgPath); statErr != nil {
		t.Fatalf("dest config.toml missing: %v", statErr)
	}
	if got, ok := out.ConfigWritten["model"]; !ok || got != "migrated" {
		t.Fatalf("ConfigWritten[model] = %q ok=%v, want migrated", got, ok)
	}

	// Env: dest .env contains importable telegram token bytes.
	envBody, envErr := os.ReadFile(envFile)
	if envErr != nil {
		t.Fatalf("read dest .env: %v", envErr)
	}
	if !strings.Contains(string(envBody), "GORMES_TELEGRAM_BOT_TOKEN=plain-telegram-token") {
		t.Fatalf("dest .env missing GORMES_TELEGRAM_BOT_TOKEN: %s", envBody)
	}
	if got, ok := out.EnvWritten["GORMES_TELEGRAM_BOT_TOKEN"]; !ok || got != "migrated" {
		t.Fatalf("EnvWritten[GORMES_TELEGRAM_BOT_TOKEN] = %q ok=%v, want migrated", got, ok)
	}

	// Memory: MEMORY.md and USER.md copied under DestMemoryDir.
	if _, statErr := os.Stat(filepath.Join(memoryDir, "MEMORY.md")); statErr != nil {
		t.Fatalf("memory MEMORY.md missing: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(memoryDir, "USER.md")); statErr != nil {
		t.Fatalf("memory USER.md missing: %v", statErr)
	}
	if got := out.MemoryWritten["memory"]; got != "migrated" {
		t.Fatalf("MemoryWritten[memory] = %q, want migrated; outcome=%+v", got, out)
	}

	// Skills: at least one skill mirrored under DestSkillsDir.
	if _, statErr := os.Stat(filepath.Join(skillsDir, "openclaw-imports", "demo", "SKILL.md")); statErr != nil {
		t.Fatalf("skills demo/SKILL.md missing: %v", statErr)
	}
	if got := out.SkillWritten["skills"]; got != "migrated" {
		t.Fatalf("SkillWritten[skills] = %q, want migrated; outcome=%+v", got, out)
	}

	// Report dir: report.json lands under ReportRootDir/<timestamp>/.
	if out.ReportPath == "" {
		t.Fatalf("ReportPath empty; outcome=%+v", out)
	}
	if !strings.HasPrefix(out.ReportPath, reportRoot) {
		t.Fatalf("ReportPath %q must start with reportRoot %q", out.ReportPath, reportRoot)
	}
	if _, statErr := os.Stat(out.ReportPath); statErr != nil {
		t.Fatalf("report file missing: %v", statErr)
	}

	// Counts: at least 1 migrated for each surface.
	if out.Counts.Migrated < 4 {
		t.Fatalf("expected migrated >= 4, got %+v", out.Counts)
	}
}

func TestOpenClawMigrationApply_ReportDirUnderXDG(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "openclaw-src")
	manifest, srcBytes := writeApplyFixture(t, src, nil)
	cfgDir, envFile, skillsDir, memoryDir, reportRoot := applyDestPaths(t, root)

	stamp := time.Date(2026, 4, 29, 9, 30, 15, 0, time.UTC)
	out, err := ApplyManifest(ApplyRequest{
		Manifest:          manifest,
		DestConfigDir:     cfgDir,
		DestEnvFile:       envFile,
		DestSkillsDir:     skillsDir,
		DestMemoryDir:     memoryDir,
		ReportRootDir:     reportRoot,
		SourceConfigBytes: srcBytes,
		SourceRoot:        src,
		SecretsEnabled:    true,
		Yes:               true,
		Now:               func() time.Time { return stamp },
	})
	if err != nil {
		t.Fatalf("ApplyManifest: %v", err)
	}
	wantPrefix := filepath.Join(reportRoot, "20260429T093015")
	if !strings.HasPrefix(out.ReportPath, wantPrefix) {
		t.Fatalf("ReportPath %q does not begin with timestamp dir %q", out.ReportPath, wantPrefix)
	}
	// Report path must live under the report root, not /tmp/random.
	if !strings.HasPrefix(out.ReportPath, reportRoot+string(os.PathSeparator)) {
		t.Fatalf("ReportPath %q is not under reportRoot %q", out.ReportPath, reportRoot)
	}
}

func TestOpenClawMigrationApply_ConflictRequiresOverwrite(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "openclaw-src")
	existing := map[string]string{
		"GORMES_TELEGRAM_BOT_TOKEN": "preset-tg-value",
	}
	manifest, srcBytes := writeApplyFixture(t, src, existing)
	cfgDir, envFile, skillsDir, memoryDir, reportRoot := applyDestPaths(t, root)

	out, err := ApplyManifest(ApplyRequest{
		Manifest:          manifest,
		DestConfigDir:     cfgDir,
		DestEnvFile:       envFile,
		DestSkillsDir:     skillsDir,
		DestMemoryDir:     memoryDir,
		ReportRootDir:     reportRoot,
		ExistingGormesEnv: existing,
		SourceConfigBytes: srcBytes,
		SourceRoot:        src,
		SecretsEnabled:    true,
		Yes:               true,
		Now:               fixedNow(t),
		// Overwrite intentionally false
	})
	if err != nil {
		t.Fatalf("ApplyManifest: %v", err)
	}
	if got := out.EnvWritten["GORMES_TELEGRAM_BOT_TOKEN"]; got != "conflict_skipped" {
		t.Fatalf("conflict without overwrite: GORMES_TELEGRAM_BOT_TOKEN = %q, want conflict_skipped; outcome=%+v", got, out)
	}
	if out.Counts.ConflictSkipped < 1 {
		t.Fatalf("Counts.ConflictSkipped = %d, want >=1", out.Counts.ConflictSkipped)
	}

	// Now retry with --overwrite. Expect migrated + a backup.
	root2 := t.TempDir()
	src2 := filepath.Join(root2, "openclaw-src")
	manifest2, srcBytes2 := writeApplyFixture(t, src2, existing)
	cfgDir2, envFile2, skillsDir2, memoryDir2, reportRoot2 := applyDestPaths(t, root2)

	// Pre-seed dest .env so backup logic engages.
	if err := os.WriteFile(envFile2, []byte("GORMES_TELEGRAM_BOT_TOKEN=preset-tg-value\n"), 0o600); err != nil {
		t.Fatalf("seed dest env: %v", err)
	}

	out2, err := ApplyManifest(ApplyRequest{
		Manifest:          manifest2,
		DestConfigDir:     cfgDir2,
		DestEnvFile:       envFile2,
		DestSkillsDir:     skillsDir2,
		DestMemoryDir:     memoryDir2,
		ReportRootDir:     reportRoot2,
		ExistingGormesEnv: existing,
		SourceConfigBytes: srcBytes2,
		SourceRoot:        src2,
		SecretsEnabled:    true,
		Yes:               true,
		Overwrite:         true,
		Now:               fixedNow(t),
	})
	if err != nil {
		t.Fatalf("ApplyManifest with overwrite: %v", err)
	}
	if got := out2.EnvWritten["GORMES_TELEGRAM_BOT_TOKEN"]; got != "migrated" {
		t.Fatalf("with --overwrite: GORMES_TELEGRAM_BOT_TOKEN = %q, want migrated; outcome=%+v", got, out2)
	}
	if len(out2.Backups) == 0 {
		t.Fatalf("expected at least one backup with overwrite, got none; outcome=%+v", out2)
	}
}

func TestOpenClawMigrationApply_SecretsRedactedWhenDisabled(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "openclaw-src")
	manifest, srcBytes := writeApplyFixture(t, src, nil)
	cfgDir, envFile, skillsDir, memoryDir, reportRoot := applyDestPaths(t, root)

	out, err := ApplyManifest(ApplyRequest{
		Manifest:          manifest,
		DestConfigDir:     cfgDir,
		DestEnvFile:       envFile,
		DestSkillsDir:     skillsDir,
		DestMemoryDir:     memoryDir,
		ReportRootDir:     reportRoot,
		SourceConfigBytes: srcBytes,
		SourceRoot:        src,
		SecretsEnabled:    false,
		Yes:               true,
		Now:               fixedNow(t),
	})
	if err != nil {
		t.Fatalf("ApplyManifest: %v", err)
	}

	// Dest .env must not contain any secret material when secrets disabled.
	envBody, _ := os.ReadFile(envFile)
	for _, leak := range []string{
		fakeOpenClawAPIKey,
		"plain-telegram-token",
		"plain-discord-token",
	} {
		if strings.Contains(string(envBody), leak) {
			t.Fatalf("dest .env leaked secret %q with SecretsEnabled=false: %s", leak, envBody)
		}
	}

	// Outcome should record at least one secret_skipped.
	if out.Counts.SecretSkipped < 1 {
		t.Fatalf("Counts.SecretSkipped = %d, want >=1; outcome=%+v", out.Counts.SecretSkipped, out)
	}

	// Report file must not contain the secret bytes.
	if out.ReportPath == "" {
		t.Fatalf("ReportPath empty: %+v", out)
	}
	reportBody, err := os.ReadFile(out.ReportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	for _, leak := range []string{
		fakeOpenClawAPIKey,
		"plain-telegram-token",
		"plain-discord-token",
	} {
		if strings.Contains(string(reportBody), leak) {
			t.Fatalf("report leaked secret %q: %s", leak, reportBody)
		}
	}

	// Outcome JSON must not leak either.
	raw, _ := json.Marshal(out)
	for _, leak := range []string{
		fakeOpenClawAPIKey,
		"plain-telegram-token",
		"plain-discord-token",
	} {
		if strings.Contains(string(raw), leak) {
			t.Fatalf("outcome JSON leaked %q: %s", leak, raw)
		}
	}
}

// stubProcessDetector is a fake ProcessDetector that returns canned
// process descriptions without invoking pgrep/systemctl.
type stubProcessDetector struct {
	procs []string
	err   error
}

func (s stubProcessDetector) Running(_ context.Context, _ string) ([]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	return append([]string(nil), s.procs...), nil
}

func TestOpenClawCleanup_DryRunPreviewsArchiveNames(t *testing.T) {
	home := t.TempDir()
	for _, dir := range []string{".openclaw", ".clawdbot", ".moltbot"} {
		if err := os.MkdirAll(filepath.Join(home, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(home, dir, "marker"), []byte("x"), 0o600); err != nil {
			t.Fatalf("seed marker: %v", err)
		}
	}

	out, err := PerformCleanup(CleanupRequest{
		HomeDir: home,
		DryRun:  true,
		Now:     fixedNow(t),
	})
	if err != nil {
		t.Fatalf("PerformCleanup dry-run: %v", err)
	}
	if !out.DryRun {
		t.Fatalf("DryRun flag must be propagated true on outcome")
	}
	if len(out.Renamed) != 3 {
		t.Fatalf("expected 3 preview rename pairs, got %d: %+v", len(out.Renamed), out.Renamed)
	}
	want := map[string]string{
		filepath.Join(home, ".openclaw"): filepath.Join(home, ".openclaw.pre-migration"),
		filepath.Join(home, ".clawdbot"): filepath.Join(home, ".clawdbot.pre-migration"),
		filepath.Join(home, ".moltbot"):  filepath.Join(home, ".moltbot.pre-migration"),
	}
	got := map[string]string{}
	for _, r := range out.Renamed {
		got[r.From] = r.To
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("dry-run rename mapping for %s = %q, want %q; outcome=%+v", k, got[k], v, out)
		}
	}
	// Dry-run must NOT rename anything on disk.
	for _, dir := range []string{".openclaw", ".clawdbot", ".moltbot"} {
		if _, err := os.Stat(filepath.Join(home, dir)); err != nil {
			t.Fatalf("dry-run should leave %s untouched: %v", dir, err)
		}
		if _, err := os.Stat(filepath.Join(home, dir+".pre-migration")); err == nil {
			t.Fatalf("dry-run must not create %s", dir+".pre-migration")
		}
	}
}

func TestOpenClawCleanup_YesRenamesNoDelete(t *testing.T) {
	home := t.TempDir()
	for _, dir := range []string{".openclaw", ".clawdbot", ".moltbot"} {
		if err := os.MkdirAll(filepath.Join(home, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		if err := os.WriteFile(filepath.Join(home, dir, "marker"), []byte("x"), 0o600); err != nil {
			t.Fatalf("seed marker: %v", err)
		}
	}

	out, err := PerformCleanup(CleanupRequest{
		HomeDir: home,
		DryRun:  false,
		Now:     fixedNow(t),
	})
	if err != nil {
		t.Fatalf("PerformCleanup: %v", err)
	}
	if out.DryRun {
		t.Fatalf("DryRun must be false on outcome")
	}
	if len(out.Renamed) != 3 {
		t.Fatalf("expected 3 renames, got %d: %+v", len(out.Renamed), out.Renamed)
	}
	for _, dir := range []string{".openclaw", ".clawdbot", ".moltbot"} {
		// Original directory must be gone (renamed).
		if _, err := os.Stat(filepath.Join(home, dir)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected %s to be renamed away, stat=%v", dir, err)
		}
		// Renamed directory exists with marker preserved (no delete).
		archived := filepath.Join(home, dir+".pre-migration")
		if _, err := os.Stat(archived); err != nil {
			t.Fatalf("expected archived %s to exist: %v", archived, err)
		}
		if _, err := os.Stat(filepath.Join(archived, "marker")); err != nil {
			t.Fatalf("expected marker preserved under %s: %v", archived, err)
		}
	}
}

func TestOpenClawCleanup_ProcessWarningInjected(t *testing.T) {
	home := t.TempDir()
	for _, dir := range []string{".openclaw"} {
		if err := os.MkdirAll(filepath.Join(home, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	det := stubProcessDetector{procs: []string{"pid 4242 openclaw-bot"}}

	out, err := PerformCleanup(CleanupRequest{
		HomeDir:  home,
		DryRun:   true,
		Detector: det,
		Now:      fixedNow(t),
	})
	if err != nil {
		t.Fatalf("PerformCleanup: %v", err)
	}
	if len(out.Warnings) == 0 {
		t.Fatalf("expected detector to inject warning, got %+v", out)
	}
	joined := strings.Join(out.Warnings, "|")
	if !strings.Contains(joined, "pid 4242 openclaw-bot") {
		t.Fatalf("warning must reference process descriptor; got %q", joined)
	}
}
