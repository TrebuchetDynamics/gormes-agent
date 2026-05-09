package installtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWindowsInstallerReleaseFirstContract pins the Windows side of the
// single-binary release promise: install.ps1 should default to a verified
// GitHub release archive on main, then fall back to source-build only when the
// operator asks for it or release fetch/verification fails.
func TestWindowsInstallerReleaseFirstContract(t *testing.T) {
	root := repoRoot(t)
	body, err := os.ReadFile(filepath.Join(root, "scripts", "install.ps1"))
	if err != nil {
		t.Fatalf("read scripts/install.ps1: %v", err)
	}
	src := string(body)

	wantAll := []string{
		"[switch]$FromSource",
		"GORMES_INSTALL_FROM_SOURCE",
		"GORMES_RELEASES_API_URL",
		"GORMES_RELEASES_DOWNLOAD_BASE",
		"$Script:InstallMethod",
		"$Script:InstallMethodDetail",
		"$Script:ReleaseArch",
		"function Get-ReleaseArch",
		"'AMD64' { 'windows-amd64' }",
		"'ARM64' { 'windows-arm64' }",
		"function Resolve-InstallMethod",
		"install_method: $($Script:InstallMethod)",
		"install_method_reason: $($Script:InstallMethodDetail)",
		"release_arch: $($Script:ReleaseArch)",
		"release_api: $($Script:GormesReleasesApiUrl)",
		"function Install-ReleaseBinary",
		".tar.gz.sha256",
		"Get-FileSha256",
		"SHA-256 mismatch",
		"tar -xzf",
		"Copy-Item -Path $extractedBin -Destination $buildBin -Force",
	}
	for _, want := range wantAll {
		if !strings.Contains(src, want) {
			t.Errorf("scripts/install.ps1 missing release-first contract fragment %q", want)
		}
	}

	installRelease := powerShellFunctionBlock(t, src, "function Install-ReleaseBinary")
	for _, want := range []string{
		"Invoke-WebRequest -Uri $archiveUrl -OutFile $archivePath",
		"Invoke-WebRequest -Uri $shaUrl -OutFile $shaPath",
		"$expected =",
		"$actual = Get-FileSha256 $archivePath",
		"if ($expected -ne $actual)",
		"return $false",
	} {
		if !strings.Contains(installRelease, want) {
			t.Errorf("Install-ReleaseBinary missing %q\nfunction body:\n%s", want, installRelease)
		}
	}

	invokeMain := powerShellFunctionBlock(t, src, "function Invoke-Main")
	for _, want := range []string{
		"Resolve-InstallMethod",
		"if ($Script:InstallMethod -eq 'binary-fetch')",
		"Install-ReleaseBinary",
		"binary-fetch failed; falling back to source build",
		"Ensure-Git",
		"Ensure-Go",
		"Install-Repository",
		"Build-Gormes",
	} {
		if !strings.Contains(invokeMain, want) {
			t.Errorf("Invoke-Main missing %q\nfunction body:\n%s", want, invokeMain)
		}
	}
	assertFragmentOrder(t, invokeMain, "Resolve-InstallMethod", "Install-ReleaseBinary")
	assertFragmentOrder(t, invokeMain, "Install-ReleaseBinary", "Ensure-Git")
	assertFragmentOrder(t, invokeMain, "Ensure-Git", "Build-Gormes")
}

func powerShellFunctionBlock(t *testing.T, src, marker string) string {
	t.Helper()
	start := strings.Index(src, marker)
	if start < 0 {
		t.Fatalf("%s not found in scripts/install.ps1", marker)
	}
	rest := src[start+len(marker):]
	end := strings.Index(rest, "\nfunction ")
	if end < 0 {
		return src[start:]
	}
	return src[start : start+len(marker)+end]
}

func assertFragmentOrder(t *testing.T, src, before, after string) {
	t.Helper()
	beforeIndex := strings.Index(src, before)
	if beforeIndex < 0 {
		t.Fatalf("missing earlier fragment %q", before)
	}
	afterIndex := strings.Index(src, after)
	if afterIndex < 0 {
		t.Fatalf("missing later fragment %q", after)
	}
	if beforeIndex >= afterIndex {
		t.Fatalf("fragment %q must appear before %q", before, after)
	}
}
