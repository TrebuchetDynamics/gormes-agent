package installtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstall_SkipSetup_DryRunReportsManualEnable proves that under
// --skip-setup the dry-run plan calls out that the gateway service file is
// installed but NOT auto-enabled. Background: the wizard is what writes
// [hermes].endpoint, so enabling the unit before setup completes guarantees
// a `provider setup: hermes endpoint unconfigured` crash-loop on next boot.
//
// The plan must still report `service    install` so the iso-bin
// boundary tests stay green; the new contract is that the line carries a
// "not auto-enabled" qualifier under --skip-setup.
func TestInstall_SkipSetup_DryRunReportsManualEnable(t *testing.T) {
	sb := t.TempDir()
	out := runInstallDryRun(t, map[string]string{
		"GORMES_INSTALL_HOME":           filepath.Join(sb, "home"),
		"GORMES_SKIP_SETUP":             "1",
		"GORMES_RESTART_GATEWAY":        "never",
		"GORMES_INSTALL_TEST_SYSTEMD":   "1",
		"GORMES_INSTALL_TEST_CONTAINER": "0",
	})
	if !strings.Contains(out, "service    install") {
		t.Fatalf("dry-run plan should still report service    install\nso fresh-user installs keep auto-start; got:\n%s", out)
	}
	if !strings.Contains(out, "not auto-enabled") {
		t.Fatalf("dry-run plan should disclose that --skip-setup leaves the\nsystemd service NOT auto-enabled; got:\n%s", out)
	}
}

// TestInstall_SystemdUnit_EnableGatedOnSetup proves the runtime side of the
// same contract: install.sh's install_systemd_user_service only calls
// `systemctl --user enable` when RUN_SETUP=true. Without this gate, a
// freshly-installed unit with no [hermes].endpoint will crash-loop on next
// login (we observed restart counters >13k in production).
func TestInstall_SystemdUnit_EnableGatedOnSetup(t *testing.T) {
	root := repoRoot(t)
	body, err := os.ReadFile(filepath.Join(root, "install.sh"))
	if err != nil {
		t.Fatalf("read install.sh: %v", err)
	}
	src := string(body)

	// Find the install_systemd_user_service function body.
	startMarker := "install_systemd_user_service() {"
	endMarker := "\n}\n"
	startIdx := strings.Index(src, startMarker)
	if startIdx < 0 {
		t.Fatal("install_systemd_user_service() not found in install.sh")
	}
	tail := src[startIdx:]
	endIdx := strings.Index(tail, endMarker)
	if endIdx < 0 {
		t.Fatal("install_systemd_user_service() body terminator not found")
	}
	body2 := tail[:endIdx]

	if !strings.Contains(body2, "systemctl --user enable") {
		t.Fatal("install_systemd_user_service() no longer calls systemctl --user enable; coverage is stale")
	}
	// The enable call must be inside an `if [ "$RUN_SETUP" = "true" ]`-style
	// guard. We accept either RUN_SETUP or a wrapper variable that derives
	// from it, but the function body must mention RUN_SETUP near the enable.
	if !strings.Contains(body2, "RUN_SETUP") {
		t.Fatal("install_systemd_user_service() must gate `systemctl --user enable` on RUN_SETUP so --skip-setup does not auto-enable a misconfigured unit")
	}
}
