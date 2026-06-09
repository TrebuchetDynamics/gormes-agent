package installtest

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstall_PrintSummaryHermesStyleChromeAndNextSteps(t *testing.T) {
	root := repoRoot(t)
	home := filepath.Join(t.TempDir(), "home")
	bin := filepath.Join(t.TempDir(), "bin")

	cmd := exec.Command("sh", "-c", ". "+shellQuote(filepath.Join(root, "install.sh"))+"; print_summary")
	cmd.Dir = root
	cmd.Env = []string{
		"HOME=" + home,
		"PATH=/usr/bin:/bin",
		"SHELL=/bin/bash",
		"GORMES_INSTALL_TEST_MODE=1",
		"GORMES_INSTALL_HOME=" + home,
		"GORMES_BIN_DIR=" + bin,
		"GORMES_SKIP_SERVICE=1",
		"GORMES_SKIP_SETUP=1",
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("print_summary failed: %v\noutput:\n%s", err, string(out))
	}

	summary := string(out)
	for _, want := range []string{
		"✓ Gormes installed",
		"Files",
		"auth",
		"log",
		"Try next",
		"gormes setup",
		"Next steps",
		"gormes doctor --offline",
		"gormes navivox pair",
		"Web",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("summary missing %q\nsummary:\n%s", want, summary)
		}
	}
}
