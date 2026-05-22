package installtest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallScriptHeaderLeadsWithReleasePOSIXShellCommand(t *testing.T) {
	root := repoRoot(t)
	installSH := readFileFromRoot(t, root, "install.sh")
	header, _, ok := strings.Cut(installSH, "\nset -eu\n")
	if !ok {
		t.Fatal("install.sh header terminator not found")
	}

	const canonical = "#   curl -fsSL https://github.com/TrebuchetDynamics/gormes-agent/releases/latest/download/install.sh | sh"
	if !strings.Contains(header, canonical) {
		t.Fatalf("install.sh header must lead with canonical POSIX shell install command %q\nheader:\n%s", canonical, header)
	}
	if strings.Contains(header, "releases/latest/download/install.sh | bash") {
		t.Fatalf("install.sh header must not advertise bash for the POSIX installer; use sh instead\nheader:\n%s", header)
	}
}

func TestInstallScriptHeaderDocumentsSourceFallbackOverride(t *testing.T) {
	root := repoRoot(t)
	installSH := readFileFromRoot(t, root, "install.sh")
	header, _, ok := strings.Cut(installSH, "\nset -eu\n")
	if !ok {
		t.Fatal("install.sh header terminator not found")
	}

	for _, want := range []string{
		"# install.sh - release-first Unix installer for Gormes, with source fallback.",
		"#   sh install.sh --from-source",
		"#   GORMES_INSTALL_FROM_SOURCE set to 1/true/yes/on to force source build",
	} {
		if !strings.Contains(header, want) {
			t.Fatalf("install.sh header missing %q\nheader:\n%s", want, header)
		}
	}
}

func TestInstallScriptHelpLeadsWithReleaseInstallerAndSourceFallback(t *testing.T) {
	root := repoRoot(t)
	cmd := exec.Command("sh", filepath.Join(root, "install.sh"), "--help")
	cmd.Dir = root
	cmd.Env = []string{"HOME=" + t.TempDir(), "PATH=/usr/bin:/bin"}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install.sh --help failed: %v\noutput:\n%s", err, string(out))
	}
	help := string(out)

	const releaseInstall = "curl -fsSL https://github.com/TrebuchetDynamics/gormes-agent/releases/latest/download/install.sh | sh"
	for _, want := range []string{
		"Gormes Unix installer",
		"Release install:",
		releaseInstall,
		"Default installs fetch the latest signed release binary",
		"--from-source",
		"GORMES_INSTALL_FROM_SOURCE=1",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("install.sh --help missing %q\nhelp:\n%s", want, help)
		}
	}
	if strings.Contains(help, "releases/latest/download/install.sh | bash") {
		t.Fatalf("install.sh --help must not advertise bash for the POSIX installer; use sh instead\nhelp:\n%s", help)
	}
}

func TestPublicInstallSurfacesLeadWithReleaseInstaller(t *testing.T) {
	root := repoRoot(t)
	const releaseInstall = "curl -fsSL https://github.com/TrebuchetDynamics/gormes-agent/releases/latest/download/install.sh | sh"
	const bannedDomainInstall = "https://gormes.ai/" + "install.sh"

	linuxDocs := readFileFromRoot(t, root, "webpages/docs/content/install/linux-macos.md")
	if !strings.Contains(linuxDocs, releaseInstall) {
		t.Fatalf("Linux/macOS install docs must expose the canonical release install command %q", releaseInstall)
	}
	if strings.Contains(linuxDocs, bannedDomainInstall) {
		t.Fatalf("Linux/macOS install docs must not reference %s", bannedDomainInstall)
	}

	landing := readFileFromRoot(t, root, "webpages/landing/src/data/landing.js")
	installIdx := strings.Index(landing, releaseInstall)
	if installIdx < 0 {
		t.Fatalf("landing install copy must expose the canonical release install command %q", releaseInstall)
	}
	sourceIdx := strings.Index(landing, "CGO_ENABLED=0 go build")
	if sourceIdx >= 0 && sourceIdx < installIdx {
		t.Fatalf("landing install copy must lead with release install before source build")
	}
	for _, reject := range []string{
		"make build",
		bannedDomainInstall,
		"raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh",
		"go install github.com/TrebuchetDynamics/gormes-agent",
	} {
		if strings.Contains(landing, reject) {
			t.Fatalf("landing install copy contains stale install/build command %q", reject)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "webpages/landing/public/install.sh")); err == nil {
		t.Fatalf("webpages/landing/public/install.sh must not exist; use GitHub Releases for the Unix installer")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat webpages/landing/public/install.sh: %v", err)
	}

	for _, rel := range []string{
		"README.md",
		"install.sh",
		".github/workflows/deploy-gormes-docs.yml",
		".github/workflows/deploy-gormes-www.yml",
		".github/workflows/release.yml",
		"webpages/docs/content/_index.md",
		"webpages/docs/content/start-here/_index.md",
		"webpages/docs/content/building-gormes/architecture_plan/progress.json",
		"webpages/landing/README.md",
		"webpages/landing/scripts/sync-assets.mjs",
		"webpages/landing/tests/home.spec.mjs",
		"webpages/landing/legacy/go-renderer/internal/site/content.go",
	} {
		body := readFileFromRoot(t, root, rel)
		if strings.Contains(body, bannedDomainInstall) {
			t.Fatalf("%s must not reference %s", rel, bannedDomainInstall)
		}
	}
}
