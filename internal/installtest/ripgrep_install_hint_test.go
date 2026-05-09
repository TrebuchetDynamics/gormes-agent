package installtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstall_RipgrepWarning_OffersDistroInstallHint guards a small UX bug
// in install.sh: when ripgrep is missing, the warning prints
// "ripgrep not found (file search will use slower fallbacks)" with no hint
// for how to install it. install.sh has already detected the distro by the
// time the check runs (see detect_os) so it can offer a one-line apt/dnf/
// brew/pacman command on supported platforms.
//
// We don't try to dictate every distro's package name here; we just require
// that check_ripgrep_optional surfaces a per-distro install hint when ripgrep
// is missing. Operators on Ubuntu/Debian especially benefit because the
// package name (ripgrep) is the same as the binary name (rg) — easy to miss.
func TestInstall_RipgrepWarning_OffersDistroInstallHint(t *testing.T) {
	root := repoRoot(t)
	body, err := os.ReadFile(filepath.Join(root, "install.sh"))
	if err != nil {
		t.Fatalf("read install.sh: %v", err)
	}
	src := string(body)

	startMarker := "check_ripgrep_optional() {"
	endMarker := "\n}\n"
	startIdx := strings.Index(src, startMarker)
	if startIdx < 0 {
		t.Fatal("check_ripgrep_optional() not found in install.sh")
	}
	tail := src[startIdx:]
	endIdx := strings.Index(tail, endMarker)
	if endIdx < 0 {
		t.Fatal("check_ripgrep_optional() body terminator not found")
	}
	fnBody := tail[:endIdx]

	// The function must reference the install command for at least one of
	// the major Linux distros + Homebrew, gated on the detected DISTRO.
	mustMention := []string{
		"DISTRO",
		"apt", // ubuntu/debian
	}
	for _, want := range mustMention {
		if !strings.Contains(fnBody, want) {
			t.Errorf("check_ripgrep_optional() missing distro install hint reference %q\nfunction body:\n%s", want, fnBody)
		}
	}
}
