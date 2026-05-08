package install_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReleaseWorkflowContract keeps the pre-compiled binary path honest. It
// verifies that the GitHub release workflow emits static Go archives for the
// product platforms Gormes claims, writes SHA-256 checksums beside them, and
// publishes only on tagged releases.
func TestReleaseWorkflowContract(t *testing.T) {
	workflow := readRepoFileRelease(t, ".github/workflows/release.yml")

	wantAll := []string{
		"name: Release Binaries",
		"tags:",
		"- 'v*'",
		"workflow_dispatch:",
		"contents: write",
		"go-version: '1.25'",
		"linux",
		"darwin",
		"windows",
		"android",
		"amd64",
		"arm64",
		"CGO_ENABLED=0",
		"go build -trimpath",
		"-ldflags=\"-s -w -X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT} -X main.GitDirty=${GIT_DIRTY}\"",
		"GIT_COMMIT=\"$(git rev-parse --short HEAD 2>/dev/null || echo unknown)\"",
		"GIT_DIRTY=false",
		"GIT_DIRTY=true",
		"gormes-${VERSION}-${GOOS}-${GOARCH}",
		"tar -C dist -czf \"dist/${target}.tar.gz\"",
		"sha256sum \"dist/${target}.tar.gz\"",
		"actions/upload-artifact@v4",
		"actions/download-artifact@v4",
		"softprops/action-gh-release@v2",
		"if: startsWith(github.ref, 'refs/tags/v')",
	}
	for _, want := range wantAll {
		if !strings.Contains(workflow, want) {
			t.Errorf("release workflow missing %q", want)
		}
	}

	wantNone := []string{
		"python -m build",
		"pip install",
		"npm install",
		"playwright install",
		"CGO_ENABLED=1",
	}
	for _, banned := range wantNone {
		if strings.Contains(workflow, banned) {
			t.Errorf("release workflow contains forbidden fragment %q", banned)
		}
	}

	for _, target := range []string{
		"linux-amd64",
		"linux-arm64",
		"darwin-amd64",
		"darwin-arm64",
		"windows-amd64",
		"windows-arm64",
		"android-arm64",
	} {
		parts := strings.Split(target, "-")
		if !strings.Contains(workflow, "goos: "+parts[0]) ||
			!strings.Contains(workflow, "goarch: "+parts[1]) {
			t.Errorf("release workflow missing target %s", target)
		}
	}
}

func TestReleaseWorkflowGeneratesSBOMsWithoutPublishingFromMatrix(t *testing.T) {
	workflow := readRepoFileRelease(t, ".github/workflows/release.yml")
	sbomStep := workflowStepBlock(t, workflow, "- name: Generate SBOM")

	wantAll := []string{
		"uses: anchore/sbom-action@v0",
		"output-file: dist/gormes-${{ steps.version.outputs.version }}-${{ matrix.goos }}-${{ matrix.goarch }}.sbom.json",
		"format: spdx-json",
		"upload-artifact: false",
		"upload-release-assets: false",
	}
	for _, want := range wantAll {
		if !strings.Contains(sbomStep, want) {
			t.Errorf("Generate SBOM step missing %q", want)
		}
	}
}

func readRepoFileRelease(t *testing.T, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(raw)
}

func workflowStepBlock(t *testing.T, workflow, stepName string) string {
	t.Helper()
	start := strings.Index(workflow, stepName)
	if start < 0 {
		t.Fatalf("workflow missing step %q", stepName)
	}
	rest := workflow[start+len(stepName):]
	end := strings.Index(rest, "\n      - ")
	if end < 0 {
		return workflow[start:]
	}
	return workflow[start : start+len(stepName)+end]
}
