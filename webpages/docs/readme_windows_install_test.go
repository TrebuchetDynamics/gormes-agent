package docs_test

import (
	"strings"
	"testing"
)

// TestReadmeQuickInstall_AdvertisesWindowsInstaller pins a UX
// regression observed in the field: README's Quick Install section
// said "Native Windows is not supported. Please install WSL2 and run
// the command above." But Gormes ships a complete native Windows
// installer at scripts/install.ps1 (verified by
// install_pwsh_test.go), and webpages/landing/scripts/sync-assets.mjs
// publishes it to https://gormes.ai/install.ps1 + install.cmd.
//
// The README was lying to operators on Windows. This test pins the
// truthful contract: README must mention install.ps1 (the actual
// Windows path), and must NOT contain the stale "not supported"
// claim. WSL2 remains a documented option, not a forced one.
func TestReadmeQuickInstall_AdvertisesWindowsInstaller(t *testing.T) {
	raw := readDoc(t, "../../README.md")

	if strings.Contains(raw, "Native Windows is not supported") {
		t.Fatalf(`README still claims Windows is not supported, but scripts/install.ps1 is the documented native installer. Remove the stale "Native Windows is not supported" copy and replace it with the install.ps1 instruction.`)
	}

	// Must mention the Windows installer path. Either install.ps1 or
	// install.cmd is acceptable — both point to the same managed
	// install flow.
	if !strings.Contains(raw, "install.ps1") && !strings.Contains(raw, "install.cmd") {
		t.Fatalf(`README must document the Windows installer (scripts/install.ps1 or install.cmd) in the Quick Install section; got no reference. Operators on Windows currently can't tell the installer exists.`)
	}
}
