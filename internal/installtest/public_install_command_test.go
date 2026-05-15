package installtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
