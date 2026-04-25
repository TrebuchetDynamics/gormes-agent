package site

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	canonicalInstallPS1 = "../../../scripts/install.ps1"
	canonicalInstallCMD = "../../../scripts/install.cmd"
	embeddedInstallPS1  = "installers/install.ps1"
	embeddedInstallCMD  = "installers/install.cmd"
)

func TestInstallPS1_SiteCopyMatchesCanonicalScript(t *testing.T) {
	canonical, err := os.ReadFile(canonicalInstallPS1)
	if err != nil {
		t.Fatalf("read canonical install.ps1: %v", err)
	}
	embedded, err := os.ReadFile(embeddedInstallPS1)
	if err != nil {
		t.Fatalf("read embedded install.ps1: %v", err)
	}
	if !bytes.Equal(embedded, canonical) {
		t.Fatal("embedded install.ps1 differs from scripts/install.ps1")
	}
}

func TestInstallCMD_SiteCopyMatchesCanonicalScript(t *testing.T) {
	canonical, err := os.ReadFile(canonicalInstallCMD)
	if err != nil {
		t.Fatalf("read canonical install.cmd: %v", err)
	}
	embedded, err := os.ReadFile(embeddedInstallCMD)
	if err != nil {
		t.Fatalf("read embedded install.cmd: %v", err)
	}
	if !bytes.Equal(embedded, canonical) {
		t.Fatal("embedded install.cmd differs from scripts/install.cmd")
	}
}

func TestInstallPS1_ContainsManagedInstallContract(t *testing.T) {
	body, err := os.ReadFile(canonicalInstallPS1)
	if err != nil {
		t.Fatalf("read install.ps1: %v", err)
	}
	text := string(body)
	for _, want := range []string{
		"LOCALAPPDATA",
		"gormes-agent",
		"GORMES_INSTALL_HOME",
		"GORMES_INSTALL_DIR",
		"GORMES_BIN_DIR",
		"GORMES_BRANCH",
		"GORMES_GO_VERSION",
		"winget",
		"choco",
		"Invoke-WebRequest",
		"go.dev/dl",
		"git fetch origin",
		"git pull --ff-only",
		"git stash push --include-untracked",
		"go build -trimpath",
		"Copy-Item",
		"Move-Item",
		"SetEnvironmentVariable('Path'",
		"Invoke-Main",
		"GORMES_INSTALL_TEST_MODE",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("install.ps1 missing %q", want)
		}
	}
}

func TestInstallCMD_WrapsPowerShellInstaller(t *testing.T) {
	body, err := os.ReadFile(canonicalInstallCMD)
	if err != nil {
		t.Fatalf("read install.cmd: %v", err)
	}
	text := string(body)
	for _, want := range []string{
		"powershell",
		"ExecutionPolicy",
		"NoProfile",
		"install.ps1",
		"GORMES_PS1_URL",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("install.cmd missing %q", want)
		}
	}
}

// TestInstallPS1_LoadsInPwshIfPresent dot-sources install.ps1 in test mode and
// invokes the path helpers to confirm the script parses and exposes a managed
// install layout. Skipped when pwsh is unavailable so contributors on hosts
// without PowerShell aren't blocked.
func TestInstallPS1_LoadsInPwshIfPresent(t *testing.T) {
	pwsh, err := exec.LookPath("pwsh")
	if err != nil {
		t.Skip("pwsh not installed")
	}
	abs, err := filepath.Abs(canonicalInstallPS1)
	if err != nil {
		t.Fatalf("Abs(install.ps1): %v", err)
	}

	tmp := t.TempDir()
	cmd := exec.Command(pwsh, "-NoProfile", "-NoLogo", "-Command",
		". '"+abs+"'; "+
			"Write-Output (Get-ManagedHome); "+
			"Write-Output (Get-ManagedCheckoutDir); "+
			"Write-Output (Get-PublishedBinDir)")
	cmd.Env = append(os.Environ(),
		"GORMES_INSTALL_TEST_MODE=1",
		"GORMES_INSTALL_HOME="+filepath.Join(tmp, "gormes"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pwsh load failed: %v\n%s", err, out)
	}
	for _, want := range []string{
		filepath.Join(tmp, "gormes"),
		filepath.Join(tmp, "gormes", "gormes-agent"),
		filepath.Join(tmp, "gormes", "bin"),
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("pwsh output missing %q\n%s", want, out)
		}
	}
}
