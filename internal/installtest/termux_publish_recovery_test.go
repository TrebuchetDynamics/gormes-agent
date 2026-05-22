package installtest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestTermuxBinaryFetchPublishVerificationFallsBackToSourceBuild proves the
// latest-release failure mode is recoverable by install.sh: a downloaded
// android-arm64 binary may publish as a real $PREFIX/bin file yet still fail
// `gormes version` under termux-exec argv injection. When that happens during
// the binary-fetch path, the installer should rebuild from source and retry
// publication instead of leaving Termux with a rolled-back command.
func TestTermuxBinaryFetchPublishVerificationFallsBackToSourceBuild(t *testing.T) {
	root := repoRoot(t)
	sb := t.TempDir()
	installHome := filepath.Join(sb, "install-home")
	prefix := filepath.Join(sb, "com.termux", "files", "usr")
	buildBin := filepath.Join(installHome, "bin", "gormes")
	publishedBin := filepath.Join(prefix, "bin", "gormes")

	if err := os.MkdirAll(filepath.Dir(buildBin), 0o755); err != nil {
		t.Fatalf("mkdir managed bin dir: %v", err)
	}
	badBinary := "#!/bin/sh\n" +
		"printf '%s\\n' 'Error: unknown command /data/data/com.termux/files/usr/bin/gormes for gormes' >&2\n" +
		"exit 1\n"
	if err := os.WriteFile(buildBin, []byte(badBinary), 0o755); err != nil {
		t.Fatalf("write bad release binary fixture: %v", err)
	}

	script := "set -e; export GORMES_INSTALL_TEST_MODE=1; " +
		"export HOME=" + shellQuote(filepath.Join(sb, "home")) + "; " +
		"export GORMES_INSTALL_HOME=" + shellQuote(installHome) + "; " +
		"export TERMUX_VERSION=0.119.0; export PREFIX=" + shellQuote(prefix) + "; " +
		". " + shellQuote(filepath.Join(root, "install.sh")) + "; " +
		"update_active_command(){ :; }; ensure_path_in_shell_config(){ :; }; " +
		"ensure_source_prerequisites(){ printf '%s\\n' SOURCE_PREREQS; }; " +
		"ensure_checkout(){ printf '%s\\n' CHECKOUT; }; " +
		"build_gormes(){ bg_bin=$(managed_bin_dir)/gormes; mkdir -p \"$(parent_dir \"$bg_bin\")\"; " +
		"printf '%s\\n' '#!/bin/sh' 'if [ \"$1\" = version ]; then echo gormes version source-fallback; exit 0; fi' 'exit 0' > \"$bg_bin\"; chmod +x \"$bg_bin\"; printf '%s\\n' BUILD_GORMES; }; " +
		"INSTALL_METHOD=binary-fetch; INSTALL_METHOD_DETAIL='android-arm64 from latest release'; " +
		"publish_command_with_recovery"

	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = root
	cmd.Env = []string{"PATH=/usr/bin:/bin", "GORMES_INSTALL_TEST_MODE=1"}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("publish recovery harness failed: %v\n%s", err, out)
	}
	output := string(out)
	if !strings.Contains(output, "falling back to source build") {
		t.Fatalf("publish recovery should explain source-build fallback; got:\n%s", output)
	}
	if !strings.Contains(output, "BUILD_GORMES") {
		t.Fatalf("publish recovery should invoke source build after binary verify failure; got:\n%s", output)
	}
	versionOut, err := exec.Command(publishedBin, "version").CombinedOutput()
	if err != nil {
		t.Fatalf("published fallback binary should run version: %v\n%s", err, versionOut)
	}
	if !strings.Contains(string(versionOut), "source-fallback") {
		t.Fatalf("published binary should be source-fallback fixture, got:\n%s", versionOut)
	}
}

// TestTermuxBinaryFetchPublishRecoveryReportsSourceBuildVerificationFailure
// proves that the recovery path does not go silent if the source-build retry
// also publishes a non-runnable command. The operator should still get the
// familiar rolled-back verification error instead of a bare `set -e` exit.
func TestTermuxBinaryFetchPublishRecoveryReportsSourceBuildVerificationFailure(t *testing.T) {
	root := repoRoot(t)
	sb := t.TempDir()
	installHome := filepath.Join(sb, "install-home")
	prefix := filepath.Join(sb, "com.termux", "files", "usr")
	buildBin := filepath.Join(installHome, "bin", "gormes")
	publishedBin := filepath.Join(prefix, "bin", "gormes")

	if err := os.MkdirAll(filepath.Dir(buildBin), 0o755); err != nil {
		t.Fatalf("mkdir managed bin dir: %v", err)
	}
	badBinary := "#!/bin/sh\n" +
		"printf '%s\\n' 'Error: unknown command /data/data/com.termux/files/usr/bin/gormes for gormes' >&2\n" +
		"exit 1\n"
	if err := os.WriteFile(buildBin, []byte(badBinary), 0o755); err != nil {
		t.Fatalf("write bad release binary fixture: %v", err)
	}

	script := "set -e; export GORMES_INSTALL_TEST_MODE=1; " +
		"export HOME=" + shellQuote(filepath.Join(sb, "home")) + "; " +
		"export GORMES_INSTALL_HOME=" + shellQuote(installHome) + "; " +
		"export TERMUX_VERSION=0.119.0; export PREFIX=" + shellQuote(prefix) + "; " +
		". " + shellQuote(filepath.Join(root, "install.sh")) + "; " +
		"update_active_command(){ :; }; ensure_path_in_shell_config(){ :; }; " +
		"ensure_source_prerequisites(){ printf '%s\\n' SOURCE_PREREQS; }; " +
		"ensure_checkout(){ printf '%s\\n' CHECKOUT; }; " +
		"build_gormes(){ bg_bin=$(managed_bin_dir)/gormes; mkdir -p \"$(parent_dir \"$bg_bin\")\"; " +
		"printf '%s\\n' '#!/bin/sh' 'echo source fallback failed >&2' 'exit 1' > \"$bg_bin\"; chmod +x \"$bg_bin\"; printf '%s\\n' BUILD_GORMES; }; " +
		"INSTALL_METHOD=binary-fetch; INSTALL_METHOD_DETAIL='android-arm64 from latest release'; " +
		"publish_command_with_recovery"

	cmd := exec.Command("sh", "-c", script)
	cmd.Dir = root
	cmd.Env = []string{"PATH=/usr/bin:/bin", "GORMES_INSTALL_TEST_MODE=1"}
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("publish recovery should fail when source-build retry still cannot run version; output:\n%s", out)
	}
	output := string(out)
	if !strings.Contains(output, "falling back to source build") {
		t.Fatalf("publish recovery should explain source-build fallback; got:\n%s", output)
	}
	want := "published command verification failed for " + publishedBin + "; rolled back"
	if !strings.Contains(output, want) {
		t.Fatalf("publish recovery should report final verification failure %q; got:\n%s", want, output)
	}
	if _, statErr := os.Lstat(publishedBin); !os.IsNotExist(statErr) {
		t.Fatalf("failed source-build retry should leave no published binary, statErr=%v", statErr)
	}
}
