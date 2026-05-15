package installtest

import (
	"strings"
	"testing"
)

func TestPublicInstallSurfacesLeadWithReleaseInstaller(t *testing.T) {
	root := repoRoot(t)
	const releaseInstall = "curl -fsSL https://gormes.ai/install.sh | sh"

	linuxDocs := readFileFromRoot(t, root, "webpages/docs/content/install/linux-macos.md")
	if !strings.Contains(linuxDocs, releaseInstall) {
		t.Fatalf("Linux/macOS install docs must expose the canonical release install command %q", releaseInstall)
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
		"raw.githubusercontent.com/TrebuchetDynamics/gormes-agent/main/install.sh",
		"go install github.com/TrebuchetDynamics/gormes-agent",
	} {
		if strings.Contains(landing, reject) {
			t.Fatalf("landing install copy contains stale install/build command %q", reject)
		}
	}
}
