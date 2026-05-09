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
		"-ldflags=\"-s -w -X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT} -X main.GitDirty=${GIT_DIRTY} -X main.BuildDate=${BUILD_DATE}\"",
		"GIT_COMMIT=\"$(git rev-parse --short HEAD 2>/dev/null || echo unknown)\"",
		"GIT_DIRTY=false",
		"GIT_DIRTY=true",
		"BUILD_DATE=\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"",
		"gormes-${VERSION}-${GOOS}-${GOARCH}",
		"archive=\"dist/${target}.tar.gz\"",
		"tar -C dist -czf \"$archive\"",
		"sha256sum \"$archive\"",
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

func TestReleaseWorkflowInjectsBuildDateProvenance(t *testing.T) {
	workflow := readRepoFileRelease(t, ".github/workflows/release.yml")
	buildStep := workflowStepBlock(t, workflow, "- name: Build static binary archive")

	wantAll := []string{
		"BUILD_DATE=\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"",
		"-X main.BuildDate=${BUILD_DATE}",
	}
	for _, want := range wantAll {
		if !strings.Contains(buildStep, want) {
			t.Errorf("Build static binary archive step missing %q", want)
		}
	}

	assertWorkflowOrder(t, buildStep,
		"BUILD_DATE=\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"",
		"go build -trimpath",
	)
	assertWorkflowOrder(t, buildStep,
		"-X main.GitDirty=${GIT_DIRTY}",
		"-X main.BuildDate=${BUILD_DATE}",
	)
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

func TestReleaseWorkflowEnforcesMaxArchiveSize(t *testing.T) {
	workflow := readRepoFileRelease(t, ".github/workflows/release.yml")
	buildStep := workflowStepBlock(t, workflow, "- name: Build static binary archive")

	wantAll := []string{
		"archive=\"dist/${target}.tar.gz\"",
		"tar -C dist -czf \"$archive\" \"${target}\"",
		"sha256sum \"$archive\" > \"${archive}.sha256\"",
		"max_archive_bytes=31457280",
		"actual_archive_bytes=$(wc -c < \"$archive\" | tr -d '[:space:]')",
		"if [ \"$actual_archive_bytes\" -gt \"$max_archive_bytes\" ]; then",
		"archive exceeds 30 MiB",
		"bytes=${actual_archive_bytes}",
		"max=${max_archive_bytes}",
		"exit 1",
	}
	for _, want := range wantAll {
		if !strings.Contains(buildStep, want) {
			t.Errorf("Build static binary archive step missing %q", want)
		}
	}

	assertWorkflowOrder(t, workflow,
		"sha256sum \"$archive\" > \"${archive}.sha256\"",
		"max_archive_bytes=31457280",
	)
	assertWorkflowOrder(t, workflow,
		"max_archive_bytes=31457280",
		"actions/upload-artifact@v4",
	)
}

func TestReleaseWorkflowReleaseNotesIncludeArchiveSize(t *testing.T) {
	workflow := readRepoFileRelease(t, ".github/workflows/release.yml")
	notesStep := workflowStepBlock(t, workflow, "- name: Build release notes")

	wantAll := []string{
		"echo \"| Platform | Archive | Size | SHA-256 |\"",
		"echo \"|----------|---------|------|---------|\"",
		"size=$(wc -c < \"$f\" | tr -d '[:space:]')",
		"echo \"| ${name%.tar.gz} | [$name]($name) | \\`${size} bytes\\` | \\`${sha}\\` |\"",
		"Software Bill of Materials (SPDX JSON) is included for each platform artifact.",
		"Build provenance attestations are published to the GitHub Attestations store.",
	}
	for _, want := range wantAll {
		if !strings.Contains(notesStep, want) {
			t.Errorf("Build release notes step missing %q", want)
		}
	}

	assertWorkflowOrder(t, notesStep,
		"size=$(wc -c < \"$f\" | tr -d '[:space:]')",
		"echo \"| ${name%.tar.gz} | [$name]($name) | \\`${size} bytes\\` | \\`${sha}\\` |\"",
	)
	assertWorkflowOrder(t, notesStep,
		"echo \"| Platform | Archive | Size | SHA-256 |\"",
		"echo \"| ${name%.tar.gz} | [$name]($name) | \\`${size} bytes\\` | \\`${sha}\\` |\"",
	)
}

// TestReleaseWorkflowPublishesInstallScripts pins the regression
// observed during the v0.2.0 fresh-install probe: a curl following
// the natural URL pattern
//
//	https://github.com/.../releases/download/v0.2.0/install.sh
//
// hit 404. install.sh and install.ps1 were not GitHub release assets;
// users had to know the canonical landing-served path
// (https://gormes.ai/install.sh) to bootstrap from a tagged release.
//
// Contract: every tagged release MUST carry install.sh and
// install.ps1 alongside the platform tarballs. The publish step
// copies them out of the source checkout into dist/ before the
// softprops upload, and surfaces them in the release notes "Install"
// block so the URL pattern is discoverable without reading the
// landing site.
func TestReleaseWorkflowPublishesInstallScripts(t *testing.T) {
	workflow := readRepoFileRelease(t, ".github/workflows/release.yml")
	publishStep := workflowStepBlock(t, workflow, "- uses: softprops/action-gh-release@v2")

	wantInUploadGlob := []string{
		"install.sh",
		"install.ps1",
	}
	for _, want := range wantInUploadGlob {
		if !strings.Contains(publishStep, want) {
			t.Errorf("softprops publish step must upload %q as a release asset; current step:\n%s", want, publishStep)
		}
	}

	// Surface in release notes so the GitHub release page itself
	// documents the canonical curl URL — operators don't need to
	// already know about gormes.ai/install.sh to bootstrap.
	notesStep := workflowStepBlock(t, workflow, "- name: Build release notes")
	wantInNotes := []string{
		"install.sh",
		"install.ps1",
	}
	for _, want := range wantInNotes {
		if !strings.Contains(notesStep, want) {
			t.Errorf("release notes step must reference %q so the GitHub release page documents the install URL; current step:\n%s", want, notesStep)
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

func assertWorkflowOrder(t *testing.T, workflow, before, after string) {
	t.Helper()
	beforeIndex := strings.Index(workflow, before)
	if beforeIndex < 0 {
		t.Fatalf("workflow missing earlier fragment %q", before)
	}
	afterIndex := strings.Index(workflow, after)
	if afterIndex < 0 {
		t.Fatalf("workflow missing later fragment %q", after)
	}
	if beforeIndex >= afterIndex {
		t.Fatalf("workflow fragment %q must appear before %q", before, after)
	}
}
