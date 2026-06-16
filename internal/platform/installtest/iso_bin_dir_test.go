// Package installtest exercises install.sh behavioral contracts via the
// fast --dry-run path. The dry-run plan is treated as install.sh's testable
// public contract: every plan line corresponds to a code-path decision the
// real install would take.
//
// These tests are intentionally fast (no git clone, no go build, no systemd
// writes). End-to-end install rehearsals are the gormes-install skill's job
// and run outside `go test`.
package installtest

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// repoRoot returns the gormes-agent repo root by walking up from this test
// file. installtest sits at <repo>/internal/platform/installtest/, so two parents up.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed; cannot locate test file path")
	}
	root, err := filepath.Abs(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	return root
}

// runInstallDryRun execs `sh install.sh --dry-run` with the given env vars
// applied on top of an empty environment. Returns combined stdout+stderr.
//
// We intentionally do NOT inherit the caller's environment, so existing
// GORMES_BIN_DIR / GORMES_PREFIX from the developer machine cannot poison
// the test outcome.
func runInstallDryRun(t *testing.T, env map[string]string, extraArgs ...string) string {
	t.Helper()
	root := repoRoot(t)
	args := append([]string{filepath.Join(root, "install.sh"), "--dry-run"}, extraArgs...)
	cmd := exec.Command("sh", args...)
	cmd.Dir = root
	cmd.Env = []string{
		// Minimum env install.sh needs to resolve managed paths.
		"HOME=" + t.TempDir(),
		"PATH=" + filepath.Join("/usr", "bin") + ":/bin",
	}
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install.sh --dry-run failed: %v\noutput:\n%s", err, string(out))
	}
	return string(out)
}

// TestInstall_DryRunSandboxBinDir_ReportsActivePathUpdateSkipped proves that
// when GORMES_BIN_DIR is set to a sandbox path, the dry-run plan reports the
// active-PATH-command update as skipped — i.e., the installer will respect
// the sandbox boundary and will NOT overwrite any other gormes binary
// discovered on PATH outside the sandbox.
//
// This encodes the iso-bin-hijack fix: the operator-explicit sandbox bin dir
// is an authoritative isolation boundary.
func TestInstall_DryRunSandboxBinDir_ReportsActivePathUpdateSkipped(t *testing.T) {
	sb := t.TempDir()
	out := runInstallDryRun(t, map[string]string{
		"GORMES_INSTALL_HOME":    filepath.Join(sb, "home"),
		"GORMES_BIN_DIR":         filepath.Join(sb, "bin"),
		"GORMES_SKIP_SETUP":      "1",
		"GORMES_RESTART_GATEWAY": "never",
	})

	if !strings.Contains(out, "path       skip") {
		t.Fatalf("dry-run plan should report active-PATH-command update as skipped\nwhen GORMES_BIN_DIR is set; got:\n%s", out)
	}
	// The reason must surface so the operator sees why isolation engaged.
	if !strings.Contains(out, "GORMES_BIN_DIR") {
		t.Fatalf("dry-run plan should name the env var that engaged sandbox isolation; got:\n%s", out)
	}
}

// TestInstall_DryRunGormesPrefix_ReportsActivePathUpdateSkipped proves the
// same isolation boundary holds for the legacy GORMES_PREFIX env var, which
// resolves to <prefix>/bin via pick_bin_dir().
func TestInstall_DryRunGormesPrefix_ReportsActivePathUpdateSkipped(t *testing.T) {
	sb := t.TempDir()
	out := runInstallDryRun(t, map[string]string{
		"GORMES_INSTALL_HOME":    filepath.Join(sb, "home"),
		"GORMES_PREFIX":          sb,
		"GORMES_SKIP_SETUP":      "1",
		"GORMES_RESTART_GATEWAY": "never",
	})

	if !strings.Contains(out, "path       skip") {
		t.Fatalf("dry-run plan should report active-PATH-command update as skipped\nwhen GORMES_PREFIX is set; got:\n%s", out)
	}
}

// TestInstall_DryRunDefaultBinDir_StillUpdatesActivePathCommand proves the
// regression fence: with no sandbox env vars, the convenient upgrade-in-place
// behavior is preserved — the dry-run plan reports the active-PATH-command
// update as enabled. This guards against an over-eager fix that would break
// fresh-user installs that rely on automatic ~/.local/bin/gormes adoption.
func TestInstall_DryRunDefaultBinDir_StillUpdatesActivePathCommand(t *testing.T) {
	sb := t.TempDir()
	out := runInstallDryRun(t, map[string]string{
		"GORMES_INSTALL_HOME":    filepath.Join(sb, "home"),
		"GORMES_SKIP_SETUP":      "1",
		"GORMES_RESTART_GATEWAY": "never",
		// Deliberately NOT setting GORMES_BIN_DIR / GORMES_PREFIX.
	})

	if !strings.Contains(out, "path       publish/adopt") {
		t.Fatalf("dry-run plan should report active-PATH-command update as enabled\nwhen no sandbox env vars are set; got:\n%s", out)
	}
	if strings.Contains(out, "path       skip") {
		t.Fatalf("dry-run plan must not report skipped when no sandbox env vars are set; got:\n%s", out)
	}
}

// TestInstall_DryRunVerboseSandbox_IncludesSkipReason proves the verbose
// plan surfaces the same isolation decision so operators auditing a verbose
// install transcript can see why the active-PATH-command update was skipped.
func TestInstall_DryRunVerboseSandbox_IncludesSkipReason(t *testing.T) {
	sb := t.TempDir()
	out := runInstallDryRun(t, map[string]string{
		"GORMES_INSTALL_HOME":    filepath.Join(sb, "home"),
		"GORMES_BIN_DIR":         filepath.Join(sb, "bin"),
		"GORMES_SKIP_SETUP":      "1",
		"GORMES_RESTART_GATEWAY": "never",
		"GORMES_INSTALL_VERBOSE": "1",
	}, "--verbose")

	if !strings.Contains(out, "path       skip") {
		t.Fatalf("verbose dry-run plan should report active-PATH-command update as skipped; got:\n%s", out)
	}
	// Verbose plan should explain the sandbox boundary semantics.
	if !strings.Contains(out, "sandbox") {
		t.Fatalf("verbose dry-run plan should mention sandbox semantics in the skip reason; got:\n%s", out)
	}
}
