// Package installtest — see iso_bin_dir_test.go for the dry-run-as-public
// contract testing rationale shared with this file.
package installtest

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestInstall_DryRunSandboxBinDir_ReportsSystemServiceInstallSkipped proves
// the iso-systemd-hijack fix: when GORMES_BIN_DIR is set, the dry-run plan
// reports system-service install as skipped — i.e., the installer will not
// rewrite the operator's ~/.config/systemd/user/gormes-gateway.service to
// point ExecStart at a sandbox path that will dangle once /tmp is reaped.
func TestInstall_DryRunSandboxBinDir_ReportsSystemServiceInstallSkipped(t *testing.T) {
	sb := t.TempDir()
	out := runInstallDryRun(t, map[string]string{
		"GORMES_INSTALL_HOME":    filepath.Join(sb, "home"),
		"GORMES_BIN_DIR":         filepath.Join(sb, "bin"),
		"GORMES_SKIP_SETUP":      "1",
		"GORMES_RESTART_GATEWAY": "never",
	})

	if !strings.Contains(out, "install_system_service: skipped") {
		t.Fatalf("dry-run plan should report system-service install as skipped\nwhen GORMES_BIN_DIR is set; got:\n%s", out)
	}
	if !strings.Contains(out, "systemd/user") {
		t.Fatalf("dry-run plan should name the production systemd path that\nthe sandbox boundary protects; got:\n%s", out)
	}
}

// TestInstall_DryRunGormesPrefix_ReportsSystemServiceInstallSkipped proves
// the same isolation boundary holds for the legacy GORMES_PREFIX env var.
func TestInstall_DryRunGormesPrefix_ReportsSystemServiceInstallSkipped(t *testing.T) {
	sb := t.TempDir()
	out := runInstallDryRun(t, map[string]string{
		"GORMES_INSTALL_HOME":    filepath.Join(sb, "home"),
		"GORMES_PREFIX":          sb,
		"GORMES_SKIP_SETUP":      "1",
		"GORMES_RESTART_GATEWAY": "never",
	})

	if !strings.Contains(out, "install_system_service: skipped") {
		t.Fatalf("dry-run plan should report system-service install as skipped\nwhen GORMES_PREFIX is set; got:\n%s", out)
	}
}

// TestInstall_DryRunDefaultBinDir_StillInstallsSystemService is the regression
// fence: with no sandbox env vars, the operator-friendly auto-install of the
// gateway unit is preserved — the dry-run plan reports system-service install
// as enabled. Guards against an over-eager fix that would break fresh-user
// installs that rely on the gateway service auto-starting on login.
func TestInstall_DryRunDefaultBinDir_StillInstallsSystemService(t *testing.T) {
	sb := t.TempDir()
	out := runInstallDryRun(t, map[string]string{
		"GORMES_INSTALL_HOME":    filepath.Join(sb, "home"),
		"GORMES_SKIP_SETUP":      "1",
		"GORMES_RESTART_GATEWAY": "never",
		// Deliberately NOT setting GORMES_BIN_DIR / GORMES_PREFIX.
	})

	if !strings.Contains(out, "install_system_service: yes") {
		t.Fatalf("dry-run plan should report system-service install as enabled\nwhen no sandbox env vars are set; got:\n%s", out)
	}
	if strings.Contains(out, "install_system_service: skipped") {
		t.Fatalf("dry-run plan must not report skipped when no sandbox env vars are set; got:\n%s", out)
	}
}
