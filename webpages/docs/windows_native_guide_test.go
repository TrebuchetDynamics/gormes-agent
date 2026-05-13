package docs_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsNativeGuideMatchesInstallerContract(t *testing.T) {
	guide := readDoc(t, "content/install/windows.md")
	install := readFirstExisting(t, "../scripts/install.ps1", "../../scripts/install.ps1")

	for _, want := range []string{
		"irm https://raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/scripts/install.ps1 | iex",
		"powershell -ExecutionPolicy Bypass -File .\\install.ps1",
		"%LOCALAPPDATA%\\gormes",
		"-DryRun",
		"-Branch",
		"-InstallHome",
		"-InstallDir",
		"-BinDir",
		"-RestartGateway",
		"-NoRestart",
		"GORMES_INSTALL_HOME",
		"GORMES_INSTALL_DIR",
		"GORMES_BIN_DIR",
		"GORMES_GO_VERSION",
		"GORMES_GO_SHA256",
		"GORMES_RESTART_GATEWAY",
		"winget",
		"choco",
		"go.dev",
		"gormes doctor --offline",
		"gormes setup",
	} {
		if !strings.Contains(guide, want) {
			t.Fatalf("windows-native guide missing %q", want)
		}
	}

	for _, reject := range []string{
		"uv",
		"virtualenv",
		"Node.js 22",
		"HERMES_HOME",
		"%LOCALAPPDATA%\\hermes",
		"hermes setup",
	} {
		if strings.Contains(guide, reject) {
			t.Fatalf("windows-native guide contains Hermes/Python copy %q", reject)
		}
	}

	for _, want := range []string{
		"param(",
		"[switch]$DryRun",
		"[switch]$Local",
		"[switch]$NoRestart",
		"GORMES_INSTALL_HOME",
		"GORMES_BIN_DIR",
		"GORMES_RESTART_GATEWAY",
		"winget",
		"choco",
		"go.dev/dl",
		"doctor --offline",
	} {
		if !strings.Contains(install, want) {
			t.Fatalf("install.ps1 missing contract token %q", want)
		}
	}
}

func TestWindowsNativeGuideLinkedFromUsingGormesPages(t *testing.T) {
	installIndex := readDoc(t, "content/install/_index.md")

	if !strings.Contains(installIndex, "./windows/") && !strings.Contains(installIndex, "Windows native") {
		t.Fatalf("install index does not link or label the Windows native guide")
	}
}

func readFirstExisting(t *testing.T, rels ...string) string {
	t.Helper()
	for _, rel := range rels {
		raw, err := os.ReadFile(filepath.Join(".", rel))
		if err == nil {
			return string(raw)
		}
	}
	t.Fatalf("none of the candidate files exist: %s", strings.Join(rels, ", "))
	return ""
}
