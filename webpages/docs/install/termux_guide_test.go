package install_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinuxMacInstallGuideDocumentsTermuxReleaseFirstPath(t *testing.T) {
	guide := readRepoFileTermuxGuide(t, "webpages/docs/content/install/linux-macos.md")
	installer := readRepoFileTermuxGuide(t, "install.sh")

	for _, want := range []string{
		"## Termux on Android",
		"android-arm64",
		"$PREFIX/bin/gormes",
		"install.sh stays in the repository root",
		"curl -fsSL https://github.com/TrebuchetDynamics/gormes-agent/releases/latest/download/install.sh | sh",
		"gormes version",
		"gormes doctor --offline --json",
		"gormes config check",
		"gormes chat -q \"hello from Termux\"",
		"source build is a fallback",
		"Only source fallback or contributor builds need the build toolchain",
		"pkg install git golang clang tmux openssh curl jq sqlite",
		"tmux",
		"tmux new -s gormes-gateway",
		"termux-wake-lock",
		"gormes gateway status",
		"gormes gateway stop",
		"Android battery-optimization",
		"Termux:Boot",
		"does not install or manage Android services automatically",
	} {
		if !strings.Contains(guide, want) {
			t.Fatalf("Termux install guide missing %q", want)
		}
	}

	if !strings.Contains(installer, "is_termux()") {
		t.Fatalf("repo-root install.sh missing Termux detection helper")
	}
	if strings.Contains(guide, "https://gormes.ai/"+"install.sh") {
		t.Fatalf("Termux guide must not point Unix users at a non-canonical gormes.ai install.sh mirror")
	}
}

func TestInstallDocsWarnTermuxV020LatestReleaseCaveat(t *testing.T) {
	for _, rel := range []string{
		"webpages/docs/content/install/_index.md",
		"webpages/docs/content/install/linux-macos.md",
		"webpages/docs/content/install/termux.md",
	} {
		assertContainsTermuxV020Caveat(t, rel, readRepoFileTermuxGuide(t, rel))
	}
}

func TestLinuxMacInstallGuideDocumentsTermuxRemoteExecutionBoundary(t *testing.T) {
	guide := readRepoFileTermuxGuide(t, "webpages/docs/content/install/linux-macos.md")
	readme := readRepoFileTermuxGuide(t, "README.md")

	for _, want := range []string{
		"phone is the Gormes controller",
		"remote host is the heavy executor",
		"ssh workstation",
		"tmux new -A -s gormes-build",
		"GORMES_REMOTE_HOST",
		"remote browser automation",
		"GPU/local model inference",
		"Docker builds",
		"large `go test ./...` runs",
		"do not add a new top-level `gormes run` command",
		"gormes chat -q",
		"gormes gateway",
	} {
		if !strings.Contains(guide, want) {
			t.Fatalf("Termux remote execution guide missing %q", want)
		}
	}
	if strings.Contains(guide, "local Termux Docker daemon") {
		t.Fatalf("Termux guide must not imply Docker runs locally on Android")
	}
	if !strings.Contains(readme, "Termux can be the controller while a remote SSH host handles Docker, browser automation, GPU/local models, and large builds") {
		t.Fatalf("README missing Termux remote-execution positioning")
	}
}

func assertContainsTermuxV020Caveat(t *testing.T, rel string, doc string) {
	t.Helper()
	for _, want := range []string{
		"Termux/Android caveat",
		"live `v0.2.20` latest-release installer",
		"unknown command /data/data/com.termux/files/usr/bin/gormes for gormes",
		"fix is committed on `development`",
		"follow-up release",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("%s missing Termux v0.2.20 caveat %q", rel, want)
		}
	}
}

func readRepoFileTermuxGuide(t *testing.T, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(raw)
}
