package gormescli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestUninstall_RemovesLegacyXDGDataHomeArtifacts pins the regression
// observed on this developer's machine after upgrading across commit
// 4cc864e80 ("fix(config): use gormes home for runtime state"):
// pre-Apr-29 binaries wrote runtime state into
// `$XDG_DATA_HOME/gormes/` (memory.db, sessions/, sessions.db,
// gateway.pid, gateway_state.json, subagents/, tools/). Post-Apr-29
// binaries write to `$GORMES_HOME` instead, but `gormes uninstall`
// still only enumerates `$GORMES_HOME` — leaving the operator with
// a fully populated legacy state directory after a "complete" wipe.
//
// Contract: uninstall must enumerate the legacy path and remove it
// alongside the current home, so an operator who upgraded across the
// migration ends up with a truly clean slate.
func TestUninstall_RemovesLegacyXDGDataHomeArtifacts(t *testing.T) {
	root := t.TempDir()
	gormesHome := filepath.Join(root, "gormes-home")
	xdgData := filepath.Join(root, "xdg-data")
	if err := os.MkdirAll(gormesHome, 0o755); err != nil {
		t.Fatalf("MkdirAll(gormesHome): %v", err)
	}
	t.Setenv("GORMES_HOME", gormesHome)
	t.Setenv("XDG_DATA_HOME", xdgData)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg-config"))
	t.Setenv("HERMES_HOME", filepath.Join(root, "hermes-home"))
	// This test verifies path enumeration/removal, not the default freedesktop
	// trash integration. Force purge keeps GIO from creating gvfs-metadata under
	// the temp XDG_DATA_HOME and racing t.TempDir cleanup.
	t.Setenv("GORMES_UNINSTALL_FORCE_PURGE", "1")

	// Seed the legacy XDG_DATA_HOME/gormes/ tree exactly the way old
	// (<= 4cc864e80~1) binaries left it.
	legacy := filepath.Join(xdgData, "gormes")
	for _, sub := range []string{"sessions", "subagents", "tools"} {
		if err := os.MkdirAll(filepath.Join(legacy, sub), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", sub, err)
		}
	}
	for _, file := range []string{
		"memory.db",
		"sessions.db",
		"gateway.pid",
		"gateway_state.json",
	} {
		if err := os.WriteFile(filepath.Join(legacy, file), []byte("legacy"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", file, err)
		}
	}
	// Sanity: legacy tree exists.
	if _, err := os.Stat(legacy); err != nil {
		t.Fatalf("legacy seed failed: %v", err)
	}

	// Also seed a current-home artifact so uninstall has something to
	// remove there too — proves the legacy path is enumerated IN
	// ADDITION TO, not INSTEAD OF, the current home.
	if err := os.WriteFile(filepath.Join(gormesHome, "config.toml"), []byte("[hermes]"), 0o644); err != nil {
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

	// Legacy tree must be gone.
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("legacy XDG_DATA_HOME/gormes still exists after uninstall: stat err=%v\nstdout=%s", err, stdout.String())
	}
	// Current-home artifact also gone (regression fence: don't break
	// the existing path while adding the legacy path).
	if _, err := os.Stat(filepath.Join(gormesHome, "config.toml")); !os.IsNotExist(err) {
		t.Fatalf("current-home config.toml not removed; stat err=%v", err)
	}
}

// TestUninstallDryRun_ListsLegacyXDGDataHomeArtifacts: the dry-run
// preview must surface the legacy path so operators see what WOULD be
// removed before they pass --yes. Without this they'd run --yes blind
// and discover the legacy cleanup only after the fact.
func TestUninstallDryRun_ListsLegacyXDGDataHomeArtifacts(t *testing.T) {
	root := t.TempDir()
	gormesHome := filepath.Join(root, "gormes-home")
	xdgData := filepath.Join(root, "xdg-data")
	if err := os.MkdirAll(gormesHome, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	t.Setenv("GORMES_HOME", gormesHome)
	t.Setenv("XDG_DATA_HOME", xdgData)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg-config"))
	t.Setenv("HERMES_HOME", filepath.Join(root, "hermes-home"))
	t.Setenv("GORMES_UNINSTALL_FORCE_PURGE", "1")

	legacy := filepath.Join(xdgData, "gormes")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatalf("MkdirAll(legacy): %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "memory.db"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newUninstallCommand()
	var stdout strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("uninstall dry-run: %v\nstdout=%s", err, stdout.String())
	}
	got := stdout.String()
	if !strings.Contains(got, legacy) {
		t.Fatalf("dry-run must list legacy XDG path %q; got:\n%s", legacy, got)
	}
}
