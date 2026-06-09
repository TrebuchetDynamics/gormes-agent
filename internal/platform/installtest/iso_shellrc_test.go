// Package installtest — see iso_bin_dir_test.go for the dry-run-as-public
// contract testing rationale shared with this file.
package installtest

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestInstall_DryRunSandboxBinDir_ReportsShellRcEditsSkipped proves the
// iso-shellrc-leak fix: when GORMES_BIN_DIR is set, the dry-run plan reports
// shell rc edits as skipped — i.e., the installer will not append "export
// PATH=/tmp/gormes-install-test/.../bin" lines to ~/.bashrc, ~/.profile,
// ~/.zshrc, or fish config that become dangling cruft once /tmp is reaped.
func TestInstall_DryRunSandboxBinDir_ReportsShellRcEditsSkipped(t *testing.T) {
	sb := t.TempDir()
	out := runInstallDryRun(t, map[string]string{
		"GORMES_INSTALL_HOME":    filepath.Join(sb, "home"),
		"GORMES_BIN_DIR":         filepath.Join(sb, "bin"),
		"GORMES_SKIP_SETUP":      "1",
		"GORMES_RESTART_GATEWAY": "never",
	})

	if !strings.Contains(out, "shell rc   skip") {
		t.Fatalf("dry-run plan should report shell rc edits as skipped\nwhen GORMES_BIN_DIR is set; got:\n%s", out)
	}
	if !strings.Contains(out, ".bashrc") {
		t.Fatalf("dry-run plan should name the production shell rc files\nthat the sandbox boundary protects; got:\n%s", out)
	}
}

// TestInstall_DryRunGormesPrefix_ReportsShellRcEditsSkipped proves the
// same isolation boundary holds for the legacy GORMES_PREFIX env var.
func TestInstall_DryRunGormesPrefix_ReportsShellRcEditsSkipped(t *testing.T) {
	sb := t.TempDir()
	out := runInstallDryRun(t, map[string]string{
		"GORMES_INSTALL_HOME":    filepath.Join(sb, "home"),
		"GORMES_PREFIX":          sb,
		"GORMES_SKIP_SETUP":      "1",
		"GORMES_RESTART_GATEWAY": "never",
	})

	if !strings.Contains(out, "shell rc   skip") {
		t.Fatalf("dry-run plan should report shell rc edits as skipped\nwhen GORMES_PREFIX is set; got:\n%s", out)
	}
}

// TestInstall_DryRunDefaultBinDir_StillEditsShellRcFiles is the regression
// fence: with no sandbox env vars, fresh-user installs still get their bin
// dir adoption — the dry-run plan reports shell rc edits as enabled. Guards
// against an over-eager fix that would break operators who rely on the
// installer making `gormes` available in new login shells.
func TestInstall_DryRunDefaultBinDir_StillEditsShellRcFiles(t *testing.T) {
	sb := t.TempDir()
	out := runInstallDryRun(t, map[string]string{
		"GORMES_INSTALL_HOME":    filepath.Join(sb, "home"),
		"GORMES_SKIP_SETUP":      "1",
		"GORMES_RESTART_GATEWAY": "never",
		// Deliberately NOT setting GORMES_BIN_DIR / GORMES_PREFIX.
	})

	if !strings.Contains(out, "shell rc   update") {
		t.Fatalf("dry-run plan should report shell rc edits as enabled\nwhen no sandbox env vars are set; got:\n%s", out)
	}
	if strings.Contains(out, "shell rc   skip") {
		t.Fatalf("dry-run plan must not report skipped when no sandbox env vars are set; got:\n%s", out)
	}
}
