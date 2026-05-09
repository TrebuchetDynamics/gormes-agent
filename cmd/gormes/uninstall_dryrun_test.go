package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUninstallDryRunDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	// Create some Gormes artifacts
	os.MkdirAll(home, 0o755)
	os.WriteFile(filepath.Join(home, "config.toml"), []byte("[hermes]"), 0o644)
	os.WriteFile(filepath.Join(home, "auth.json"), []byte(`{}`), 0o644)
	os.WriteFile(filepath.Join(home, "gateway_state.json"), []byte(`{}`), 0o644)

	groups := collectArtifacts(home)

	total := 0
	for _, g := range groups {
		total += len(g.Paths)
	}
	if total < 3 {
		t.Fatalf("expected at least 3 artifact paths, got %d", total)
	}

	// Verify groups are present
	names := make(map[string]bool)
	for _, g := range groups {
		if len(g.Paths) > 0 {
			names[g.Name] = true
		}
	}
	if !names["config"] || !names["credentials"] || !names["gateway-state"] {
		t.Fatalf("expected config, credentials, gateway-state groups: %v", names)
	}

	// Verify paths are sorted
	for _, g := range groups {
		for i := 1; i < len(g.Paths); i++ {
			if g.Paths[i] < g.Paths[i-1] {
				t.Errorf("%s paths not sorted: %s < %s", g.Name, g.Paths[i], g.Paths[i-1])
			}
		}
	}
}

func TestUninstallKeepConfigExcludesGroup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	os.WriteFile(filepath.Join(home, "config.toml"), []byte("[hermes]"), 0o644)
	os.WriteFile(filepath.Join(home, "auth.json"), []byte(`{}`), 0o644)

	groups := collectArtifacts(home)
	groups = removeGroup(groups, "config")

	for _, g := range groups {
		if g.Name == "config" {
			t.Fatal("config group should be excluded")
		}
	}
}

// TestUninstallCommand_DryRunRoutesThroughCobraWriters proves the
// command writes its dry-run output via cmd.OutOrStdout() so end-to-end
// tests can capture stdout instead of forking through helpers like
// printDryRunTo. Without this, runUninstall's fmt.Printf goes to
// os.Stdout directly and bypasses cobra's writer plumbing — meaning
// the test fixture in this file had to copy-paste a parallel
// `printDryRunTo` helper to verify formatting. The refactor matches
// the same testability pattern doctor adopted (commit 39ff073db).
func TestUninstallCommand_DryRunRoutesThroughCobraWriters(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte("[hermes]"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newUninstallCommand()
	var stdout, stderr strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("uninstall: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	got := stdout.String()
	if got == "" {
		t.Fatalf("uninstall must write to cmd.OutOrStdout(); captured stdout is empty (output likely went to os.Stdout)")
	}
	for _, want := range []string{"uninstall dry-run:", "[config]", "config.toml"} {
		if !strings.Contains(got, want) {
			t.Fatalf("uninstall stdout missing %q:\n%s", want, got)
		}
	}
}

// TestUninstallExecute_ReportsRemovedAndFailedCounts proves the
// destructive path surfaces an operator-meaningful summary at the end:
// "uninstall complete: N removed, M failed (warnings above)" rather than
// the previous bare "uninstall complete." that emitted regardless of
// whether anything succeeded. Without these counts an operator who hit
// permission errors on every file would see "uninstall complete" and
// assume success — a real footgun for cleanup workflows.
func TestUninstallExecute_ReportsRemovedAndFailedCounts(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	// Two real, removable files.
	for _, name := range []string{"config.toml", "auth.json"} {
		if err := os.WriteFile(filepath.Join(home, name), []byte("body"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cmd := newUninstallCommand()
	var stdout, stderr strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dry-run=false", "--yes"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("uninstall: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "uninstall complete") {
		t.Fatalf("stdout missing completion line:\n%s", out)
	}
	if !strings.Contains(out, "removed") {
		t.Fatalf("stdout must report removed count; got:\n%s", out)
	}
	if !strings.Contains(out, "failed") {
		t.Fatalf("stdout must report failed count (even when zero); got:\n%s", out)
	}
}

// TestUninstallExecute_FailureExitsNonZero proves the destructive path
// returns a non-nil error (rendered by cobra as a non-zero process
// exit code) when at least one removal failed. Without this, fleet
// scripts running `gormes uninstall --yes && echo OK` would believe
// uninstall succeeded even when permission errors blocked every file.
// The per-file warnings are already on stderr; the exit code is the
// machine signal scripts can branch on.
func TestUninstallExecute_FailureExitsNonZero(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	// Seed a real artifact whose REMOVAL fails: a directory whose
	// PARENT is read-only via 0o500. With no write bit on the parent,
	// unlink of the inner file is denied, and so is rmdir of the
	// guarded dir itself.
	innerDir := filepath.Join(home, "memory")
	if err := os.MkdirAll(innerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(innerDir, "guarded.bin"), []byte("body"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(innerDir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// Restore mode so the t.TempDir cleanup succeeds.
		_ = os.Chmod(innerDir, 0o755)
	})

	cmd := newUninstallCommand()
	var stdout, stderr strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"--dry-run=false", "--yes"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("uninstall must return non-nil when at least one removal failed; stdout=%s\nstderr=%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "failed") {
		t.Fatalf("stdout should still report the failed count on error path:\n%s", stdout.String())
	}
	// The summary line must show a positive failed count, not zero.
	if !strings.Contains(stdout.String(), "1 failed") && !strings.Contains(stdout.String(), "2 failed") {
		t.Fatalf("expected `N failed` with N>=1 in summary; got:\n%s", stdout.String())
	}
}

// TestUninstallDryRun_SkipsEmptyGroups proves the dry-run preview
// suppresses group headers (e.g. "[cron]" or "[mcp-oauth]") when the
// group has no actual artifacts on disk. Without this, every dry-run
// shows a wall of empty bracketed headers — operators have to scan
// through them to find the groups that actually have content. Showing
// only non-empty groups keeps the preview focused on what would
// actually be removed.
func TestUninstallDryRun_SkipsEmptyGroups(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	// Seed only the config group; other groups (credentials, sessions,
	// gateway-state, memory, logs, cron, mcp-oauth) have no on-disk
	// artifacts.
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte("[hermes]"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newUninstallCommand()
	var stdout, stderr strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("uninstall: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	got := stdout.String()
	// The config group must appear (it has an artifact).
	if !strings.Contains(got, "[config]") {
		t.Fatalf("config group missing from dry-run:\n%s", got)
	}
	// Truly empty groups must not appear. (Note: the `logs` group
	// includes config.CrashLogDir() which equals GormesHome itself,
	// so it always has a non-empty Paths list under a t.TempDir
	// fixture — that's expected and not what this test checks.)
	for _, empty := range []string{"[cron]", "[mcp-oauth]", "[memory]", "[gateway-state]", "[sessions]", "[credentials]"} {
		if strings.Contains(got, empty) {
			t.Fatalf("empty group %s should be suppressed in dry-run; got:\n%s", empty, got)
		}
	}
}

func TestUninstallDryRunOutputIsGroupedAndStable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	os.WriteFile(filepath.Join(home, "config.toml"), []byte("[hermes]"), 0o644)

	// First run — exercise the real printDryRun, not a copy-paste helper.
	groups1 := collectArtifacts(home)
	var sb1 strings.Builder
	if err := printDryRun(&sb1, groups1); err != nil {
		t.Fatalf("printDryRun: %v", err)
	}

	// Second run should produce identical output.
	groups2 := collectArtifacts(home)
	var sb2 strings.Builder
	if err := printDryRun(&sb2, groups2); err != nil {
		t.Fatalf("printDryRun: %v", err)
	}

	if sb1.String() != sb2.String() {
		t.Errorf("dry-run output not byte-stable between runs")
	}
}

// TestUninstallCommand_DryRunJSONEmitsStructuredPreview proves
// `gormes uninstall --json` (default dry-run) returns a parseable
// `{build, action: "preview", dry_run: true, total, groups: [{name, paths}]}`
// document so fleet automation can audit what WOULD be removed across
// many machines without scraping bracketed prose. Same convention as
// restore --json preview.
func TestUninstallCommand_DryRunJSONEmitsStructuredPreview(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	for _, name := range []string{"config.toml", "auth.json"} {
		if err := os.WriteFile(filepath.Join(home, name), []byte("body"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cmd := newUninstallCommand()
	var stdout strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("uninstall --json (dry-run): %v\nstdout=%s", err, stdout.String())
	}

	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Action string `json:"action"`
		DryRun bool   `json:"dry_run"`
		Total  int    `json:"total"`
		Groups []struct {
			Name  string   `json:"name"`
			Paths []string `json:"paths"`
		} `json:"groups"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout.String()), &got); jsonErr != nil {
		t.Fatalf("uninstall --json (dry-run) must be valid JSON: %v\nstdout=%s", jsonErr, stdout.String())
	}
	if got.Build.Version != Version {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, Version)
	}
	if got.Action != "preview" {
		t.Errorf("action = %q, want %q", got.Action, "preview")
	}
	if !got.DryRun {
		t.Errorf("dry_run must be true in dry-run JSON")
	}
	if got.Total < 2 {
		t.Errorf("total = %d, want >=2 (config.toml + auth.json)", got.Total)
	}
	// Files MUST stay untouched — JSON dry-run is still a dry-run.
	if _, err := os.Stat(filepath.Join(home, "config.toml")); err != nil {
		t.Errorf("dry-run JSON must not delete files: config.toml stat err=%v", err)
	}
}

// TestUninstallCommand_ExecuteJSONEmitsStructuredOutcome proves
// `--dry-run=false --yes --json` returns
// `{build, action: "uninstalled", removed, failed, groups: [...]}`
// so fleet automation can confirm cleanup completed across machines.
// Apply mode reports actual removed/failed counts; dry_run is false.
func TestUninstallCommand_ExecuteJSONEmitsStructuredOutcome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	for _, name := range []string{"config.toml", "auth.json"} {
		if err := os.WriteFile(filepath.Join(home, name), []byte("body"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cmd := newUninstallCommand()
	var stdout strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"--dry-run=false", "--yes", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("uninstall --yes --json: %v\nstdout=%s", err, stdout.String())
	}

	var got struct {
		Action  string `json:"action"`
		DryRun  bool   `json:"dry_run"`
		Removed int    `json:"removed"`
		Failed  int    `json:"failed"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout.String()), &got); jsonErr != nil {
		t.Fatalf("uninstall --yes --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout.String())
	}
	if got.Action != "uninstalled" {
		t.Errorf("action = %q, want %q", got.Action, "uninstalled")
	}
	if got.DryRun {
		t.Errorf("dry_run must be false in apply mode")
	}
	if got.Removed < 2 {
		t.Errorf("removed = %d, want >=2 (config.toml + auth.json)", got.Removed)
	}
	if got.Failed != 0 {
		t.Errorf("failed = %d, want 0", got.Failed)
	}
	if _, err := os.Stat(filepath.Join(home, "config.toml")); !os.IsNotExist(err) {
		t.Errorf("apply must remove config.toml; stat err=%v", err)
	}
}

// TestUninstall_RemovesPublishedBinarySymlink pins the cleanup
// completeness gap observed during a v0.2.0 fresh-install probe:
// `gormes uninstall --yes` removed `~/.gormes/` (including
// `~/.gormes/bin/gormes`) but left the install.sh-published PATH
// symlink at `~/.local/bin/gormes` dangling. After uninstall, `which
// gormes` returns empty, but the broken symlink persists and confuses
// reinstalls and shell completions.
//
// Contract: when the published-binary symlink lives at the install.sh
// default location AND points to a path inside the gormes home (the
// install.sh-managed binary), uninstall must remove the symlink too.
// We resolve the binary location via GORMES_BIN_DIR (install.sh's
// override env var) so the check works for any install.sh layout, not
// just the $HOME/.local/bin default.
//
// Safety: only the symlink is removed, never an unrelated `gormes`
// command (e.g., one built from source, installed via package manager,
// or shadowing on PATH). The "is it a symlink whose target is inside
// the gormes home" predicate keeps the cleanup surgical.
func TestUninstall_RemovesPublishedBinarySymlink(t *testing.T) {
	home := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_BIN_DIR", binDir)

	managedBinary := filepath.Join(home, "bin", "gormes")
	if err := os.MkdirAll(filepath.Dir(managedBinary), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(managedBinary, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	publishedSymlink := filepath.Join(binDir, "gormes")
	if err := os.Symlink(managedBinary, publishedSymlink); err != nil {
		t.Fatal(err)
	}

	cmd := newUninstallCommand()
	var stdout, stderr strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dry-run=false", "--yes", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("uninstall --yes --json: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}

	// The published-binary symlink MUST be gone. Use Lstat so a broken
	// symlink (target deleted but link itself survived — the actual
	// regression) is also caught.
	if _, err := os.Lstat(publishedSymlink); !os.IsNotExist(err) {
		t.Fatalf("uninstall must remove published-binary symlink at %s; Lstat err=%v", publishedSymlink, err)
	}

	// Must surface the path under a published-binary group in the JSON
	// outcome so fleet automation can audit it.
	var got struct {
		Groups []struct {
			Name  string   `json:"name"`
			Paths []string `json:"paths"`
		} `json:"groups"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout.String()), &got); jsonErr != nil {
		t.Fatalf("uninstall --yes --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout.String())
	}
	var found bool
	for _, g := range got.Groups {
		if g.Name == "published-binary" {
			for _, p := range g.Paths {
				if p == publishedSymlink {
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatalf("published-binary group must list %s in JSON groups; got groups=%+v", publishedSymlink, got.Groups)
	}
}

// TestUninstall_HomeDirWildcardIsNotMisclassifiedAsLogs pins the
// regression observed during the v0.2.0 fresh-install probe: the
// preview JSON listed `<home>/` under the "logs" group because
// `config.CrashLogDir()` returns `GormesHome()`. Fleet automation
// reading
//
//	{"name": "logs", "paths": ["/home/xel/.gormes/"]}
//
// reasonably interpreted that as "removes log files," but
// `os.RemoveAll("/home/xel/.gormes/")` actually nukes the entire
// home tree (config, sessions, skills, subagents, kanban DB, the
// managed binary subdirectory — everything).
//
// Contract: the wholesale home-tree removal must surface under a
// dedicated "gormes-home" group with the home directory as its
// only path. The "logs" group is reserved for explicit log files
// (gormes.log) — never the home wildcard. Operators reading the
// preview see the scope honestly.
func TestUninstall_HomeDirWildcardIsNotMisclassifiedAsLogs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte("[hermes]"), 0o644); err != nil {
		t.Fatal(err)
	}

	groups := collectArtifacts(home)
	for _, g := range groups {
		for _, p := range g.Paths {
			isHomeWildcard := strings.TrimSuffix(p, "/") == strings.TrimSuffix(home, "/")
			if isHomeWildcard && g.Name == "logs" {
				t.Fatalf("home-dir wildcard %q must not be classified as `logs` — that misleads operators about scope. Found in group=%q", p, g.Name)
			}
		}
	}

	var found bool
	for _, g := range groups {
		if g.Name != "gormes-home" {
			continue
		}
		for _, p := range g.Paths {
			if strings.TrimSuffix(p, "/") == strings.TrimSuffix(home, "/") {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("home-dir wildcard %q must appear under a dedicated `gormes-home` group; got groups=%+v", home, groups)
	}
}

// TestUninstall_LeavesUnrelatedGormesBinaryAlone proves the cleanup is
// surgical: a `gormes` command on PATH that is NOT a symlink into the
// gormes home (e.g., a binary built from source, installed via apt, or
// shadowing on PATH) MUST NOT be touched. The published-binary group
// only enumerates entries we know we put there.
func TestUninstall_LeavesUnrelatedGormesBinaryAlone(t *testing.T) {
	home := t.TempDir()
	binDir := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_BIN_DIR", binDir)

	// A real file at the bin path, NOT a symlink. Could be a
	// dev-built binary an operator dropped there manually.
	unrelated := filepath.Join(binDir, "gormes")
	if err := os.WriteFile(unrelated, []byte("#!/bin/sh\necho hi\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := newUninstallCommand()
	var stdout, stderr strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--dry-run=false", "--yes"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("uninstall: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}

	if _, err := os.Stat(unrelated); err != nil {
		t.Fatalf("uninstall must not remove a non-symlink binary at %s; got err=%v", unrelated, err)
	}
}
