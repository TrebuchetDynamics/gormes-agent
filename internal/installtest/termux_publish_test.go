package installtest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// publishHarness sources install.sh in library mode (GORMES_INSTALL_TEST_MODE=1
// so main never runs), stages a fake managed binary under an $HOME-like dir,
// and calls publish_built_binary directly for the given bin dir. It returns
// the published path's `stat` classification ("symlink" or "regular"), whether
// the post-publish verify rolled back, and the resolved (symlink-followed)
// target. This is the exec-domain that the Termux v0.2.12 regression
// (known-issues [termux-publish-symlink-noexec]) lives in: Android blocks
// execve of a published symlink whose target resolves under app-writable
// $HOME, so the published command must be a real file under $PREFIX/bin.
func publishHarness(t *testing.T, termux bool) (kind, resolved string, ok bool) {
	t.Helper()
	root := repoRoot(t)
	sb := t.TempDir()
	installHome := filepath.Join(sb, "install-home") // stands in for $HOME/.gormes
	prefix := filepath.Join(sb, "com.termux", "files", "usr")
	buildBin := filepath.Join(installHome, "bin", "gormes")
	publishedBin := filepath.Join(prefix, "bin", "gormes")

	if err := os.MkdirAll(filepath.Dir(buildBin), 0o755); err != nil {
		t.Fatalf("mkdir managed bin dir: %v", err)
	}
	if err := os.WriteFile(buildBin, []byte("#!/bin/sh\necho gormes version 0.0.0-test\n"), 0o755); err != nil {
		t.Fatalf("write fake managed binary: %v", err)
	}

	termuxEnv := ""
	if termux {
		// is_termux() is true when TERMUX_VERSION is set or PREFIX matches
		// the com.termux usr path; set both to mirror a real device.
		termuxEnv = "export TERMUX_VERSION=0.119.0; export PREFIX=" + shellQuote(prefix) + ";"
	}
	script := "set -e; export GORMES_INSTALL_TEST_MODE=1; " + termuxEnv +
		". " + shellQuote(filepath.Join(root, "install.sh")) + "; " +
		"publish_built_binary " + shellQuote(buildBin) + " " + shellQuote(publishedBin) + " >/dev/null 2>&1; " +
		"echo PUBLISH_RC=$?"

	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GORMES_INSTALL_TEST_MODE=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("publish_built_binary harness failed: %v\n%s", err, out)
	}
	publishOK := strings.Contains(string(out), "PUBLISH_RC=0")

	fi, statErr := os.Lstat(publishedBin)
	if statErr != nil {
		// Published command absent → rollback fired (the v0.2.12 symptom).
		return "absent", "", publishOK
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		tgt, _ := filepath.EvalSymlinks(publishedBin)
		return "symlink", tgt, publishOK
	}
	real, _ := filepath.EvalSymlinks(publishedBin)
	return "regular", real, publishOK
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// TestTermuxPublishesRealBinaryNotHomeSymlink is the regression fence for
// known-issues [termux-publish-symlink-noexec]: on Termux the published
// $PREFIX/bin/gormes must be a real executable file, never a symlink that
// resolves under the install/$HOME tree (Android 10+ blocks execve there).
func TestTermuxPublishesRealBinaryNotHomeSymlink(t *testing.T) {
	kind, resolved, ok := publishHarness(t, true)
	if kind == "absent" {
		t.Fatalf("Termux publish rolled back the published command (the v0.2.12 symptom): no $PREFIX/bin/gormes left")
	}
	if kind == "symlink" {
		t.Fatalf("Termux published command is a symlink resolving to %q; Android blocks execve of $HOME-resolved targets. It must be a real copied file.", resolved)
	}
	if kind != "regular" {
		t.Fatalf("Termux published command kind = %q, want regular file", kind)
	}
	if strings.Contains(resolved, "install-home") {
		t.Fatalf("Termux published command resolves into the managed/$HOME tree (%q); must be a standalone $PREFIX/bin file", resolved)
	}
	if !ok {
		t.Fatalf("Termux publish verify did not succeed (expected PUBLISH_RC=0 with a real $PREFIX/bin file)")
	}
}

// TestNonTermuxPublishKeepsSymlink fences the fix so it does not change
// non-Termux behavior: the existing symlink-preferred publish path stays.
func TestNonTermuxPublishKeepsSymlink(t *testing.T) {
	kind, _, ok := publishHarness(t, false)
	if !ok {
		t.Fatalf("non-Termux publish verify did not succeed")
	}
	if kind != "symlink" {
		t.Fatalf("non-Termux published command kind = %q, want symlink (unchanged behavior)", kind)
	}
}
