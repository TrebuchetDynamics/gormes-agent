package install_test

import (
	"os"
	"strings"
	"testing"
)

// TestReleaseWorkflowContract keeps the pre-compiled binary path honest. It
// verifies that the GitHub release workflow emits static Go archives for the
// product platforms Gormes claims, writes SHA-256 checksums beside them, and
// publishes only on tagged releases.
func TestReleaseWorkflowContract(t *testing.T) {
	workflow := readRepoFile(t, ".github/workflows/release.yml")

	wantAll := []string{
		"name: Release Binaries",
		"tags:",
		"- 'v*'",
		"workflow_dispatch:",
		"contents: write",
		"actions/setup-go@v6",
		"go-version-file: go.mod",
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
		"actions/upload-artifact@v7",
		"actions/download-artifact@v8",
		"softprops/action-gh-release@v3",
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

func TestReleaseWorkflowValidateJobCoversCIBlogGate(t *testing.T) {
	workflow := readRepoFile(t, ".github/workflows/release.yml")

	wantAll := []string{
		"cache-dependency-path: |",
		"webpages/docs/package-lock.json",
		"webpages/blog/package-lock.json",
		"- name: Install docs dependencies",
		"working-directory: webpages/docs",
		"run: npm ci",
		"- name: Install blog dependencies",
		"working-directory: webpages/blog",
		"- name: Test engineering blog",
		"run: npm run test",
	}
	for _, want := range wantAll {
		if !strings.Contains(workflow, want) {
			t.Errorf("release validate job missing %q", want)
		}
	}

	installBlog := workflowStepBlock(t, workflow, "- name: Install blog dependencies")
	if !strings.Contains(installBlog, "working-directory: webpages/blog") || !strings.Contains(installBlog, "run: npm ci") {
		t.Fatalf("Install blog dependencies step is incomplete:\n%s", installBlog)
	}
	testBlog := workflowStepBlock(t, workflow, "- name: Test engineering blog")
	if !strings.Contains(testBlog, "working-directory: webpages/blog") || !strings.Contains(testBlog, "run: npm run test") {
		t.Fatalf("Test engineering blog step is incomplete:\n%s", testBlog)
	}

	assertWorkflowOrder(t, workflow, "webpages/docs/package-lock.json", "webpages/blog/package-lock.json")
	assertWorkflowOrder(t, workflow, "- name: Install docs dependencies", "- name: Install blog dependencies")
	assertWorkflowOrder(t, workflow, "- name: Install blog dependencies", "- name: Run Go tests")
	assertWorkflowOrder(t, workflow, "- name: Run Go tests", "- name: Test engineering blog")
	assertWorkflowOrder(t, workflow, "- name: Test engineering blog", "- name: Validate progress")
}

func TestReleaseWorkflowInjectsBuildDateProvenance(t *testing.T) {
	workflow := readRepoFile(t, ".github/workflows/release.yml")
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

func TestReleaseWorkflowSmokeChecksBinaryVersionMetadata(t *testing.T) {
	workflow := readRepoFile(t, ".github/workflows/release.yml")
	buildStep := workflowStepBlock(t, workflow, "- name: Build static binary archive")

	wantAll := []string{
		"binary_path=\"dist/${target}/${exe}\"",
		"if [ \"$GOOS\" = \"$(go env GOHOSTOS)\" ] && [ \"$GOARCH\" = \"$(go env GOHOSTARCH)\" ]; then",
		"version_json=$(\"$binary_path\" version --json)",
		"grep \"\\\"version\\\": \\\"${VERSION}\\\"\"",
		"grep \"\\\"git_commit\\\": \\\"${GIT_COMMIT}\\\"\"",
		"grep \"\\\"git_dirty\\\": ${GIT_DIRTY}\"",
		"grep \"\\\"build_date\\\": \\\"${BUILD_DATE}\\\"\"",
		"go version -m \"$binary_path\"",
	}
	for _, want := range wantAll {
		if !strings.Contains(buildStep, want) {
			t.Errorf("Build static binary archive step missing %q", want)
		}
	}

	assertWorkflowOrder(t, buildStep,
		"go build -trimpath",
		"version_json=$(\"$binary_path\" version --json)",
	)
	assertWorkflowOrder(t, buildStep,
		"go version -m \"$binary_path\"",
		"tar -C dist -czf \"$archive\"",
	)
	assertWorkflowOrder(t, buildStep,
		"version_json=$(\"$binary_path\" version --json)",
		"tar -C dist -czf \"$archive\"",
	)
}

func TestReleaseWorkflowGeneratesSBOMsWithoutPublishingFromMatrix(t *testing.T) {
	workflow := readRepoFile(t, ".github/workflows/release.yml")
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

func TestReleaseWorkflowAttestsSBOMsForArchives(t *testing.T) {
	workflow := readRepoFile(t, ".github/workflows/release.yml")
	sbomStep := workflowStepBlock(t, workflow, "- name: Generate SBOM")
	attestStep := workflowStepBlock(t, workflow, "- name: Attest SBOM")

	wantAll := []string{
		"uses: actions/attest@v4",
		"subject-path: dist/gormes-${{ steps.version.outputs.version }}-${{ matrix.goos }}-${{ matrix.goarch }}.tar.gz",
		"sbom-path: dist/gormes-${{ steps.version.outputs.version }}-${{ matrix.goos }}-${{ matrix.goarch }}.sbom.json",
	}
	for _, want := range wantAll {
		if !strings.Contains(attestStep, want) {
			t.Errorf("Attest SBOM step missing %q", want)
		}
	}

	assertWorkflowOrder(t, workflow, "- name: Generate SBOM", "- name: Attest SBOM")
	assertWorkflowOrder(t, workflow, "- name: Attest SBOM", "actions/upload-artifact@v7")
	assertWorkflowOrder(t, workflow, sbomStep, attestStep)
}

func TestReleaseWorkflowAttestsBuildProvenanceForArchives(t *testing.T) {
	workflow := readRepoFile(t, ".github/workflows/release.yml")
	provenanceStep := workflowStepBlock(t, workflow, "- name: Attest build provenance")

	wantAll := []string{
		"uses: actions/attest@v4",
		"subject-path: dist/gormes-${{ steps.version.outputs.version }}-${{ matrix.goos }}-${{ matrix.goarch }}.tar.gz",
	}
	for _, want := range wantAll {
		if !strings.Contains(provenanceStep, want) {
			t.Errorf("Attest build provenance step missing %q\nstep body:\n%s", want, provenanceStep)
		}
	}
	if strings.Contains(provenanceStep, "sbom-path:") {
		t.Errorf("build provenance attestation must not use sbom-path; current step:\n%s", provenanceStep)
	}

	assertWorkflowOrder(t, workflow, "- name: Attest SBOM", "- name: Attest build provenance")
	assertWorkflowOrder(t, workflow, "- name: Attest build provenance", "actions/upload-artifact@v7")
}

func TestReleaseWorkflowEnforcesMaxArchiveSize(t *testing.T) {
	workflow := readRepoFile(t, ".github/workflows/release.yml")
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
		"actions/upload-artifact@v7",
	)
}

func TestReleaseWorkflowReleaseNotesIncludeArchiveSize(t *testing.T) {
	workflow := readRepoFile(t, ".github/workflows/release.yml")
	notesStep := workflowStepBlock(t, workflow, "- name: Build release notes")

	wantAll := []string{
		"echo \"| Platform | Archive | Size | SHA-256 |\"",
		"echo \"|----------|---------|------|---------|\"",
		"size=$(wc -c < \"$f\" | tr -d '[:space:]')",
		"echo \"| ${name%.tar.gz} | [$name]($name) | \\`${size} bytes\\` | \\`${sha}\\` |\"",
		"Software Bill of Materials (SPDX JSON) is included for each platform artifact.",
		"SBOM and build-provenance attestations are published to the GitHub Attestations store.",
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

func TestReleaseWorkflowReleaseNotesNameSBOMAttestations(t *testing.T) {
	workflow := readRepoFile(t, ".github/workflows/release.yml")
	notesStep := workflowStepBlock(t, workflow, "- name: Build release notes")

	wantAll := []string{
		"echo \"## SBOM\"",
		"Software Bill of Materials (SPDX JSON) is included for each platform artifact.",
		"echo \"## Verification\"",
		"SBOM and build-provenance attestations are published to the GitHub Attestations store.",
	}
	for _, want := range wantAll {
		if !strings.Contains(notesStep, want) {
			t.Errorf("Build release notes step missing %q", want)
		}
	}

	stale := "Build provenance attestations are published to the GitHub Attestations store."
	if strings.Contains(notesStep, stale) {
		t.Errorf("release notes still imply only build provenance is attested; replace %q with SBOM + build-provenance wording", stale)
	}

	assertWorkflowOrder(t, notesStep,
		"Software Bill of Materials (SPDX JSON) is included for each platform artifact.",
		"SBOM and build-provenance attestations are published to the GitHub Attestations store.",
	)
}

func TestReleaseWorkflowReleaseTitleCarriesDateAlias(t *testing.T) {
	workflow := readRepoFile(t, ".github/workflows/release.yml")
	notesStep := workflowStepBlock(t, workflow, "- name: Build release notes")
	publishStep := workflowStepBlock(t, workflow, "- uses: softprops/action-gh-release@v3")

	wantInNotesStep := []string{
		"id: release_notes",
		"date_alias=$(grep 'var VersionDateAlias =' cmd/gormes/main.go | sed 's/.*\"\\(.*\\)\".*/\\1/')",
		"test -n \"$date_alias\"",
		"echo \"date_alias=$date_alias\" >> \"$GITHUB_OUTPUT\"",
		"echo \"# Gormes ${GITHUB_REF_NAME} (${date_alias})\"",
	}
	for _, want := range wantInNotesStep {
		if !strings.Contains(notesStep, want) {
			t.Errorf("Build release notes step missing %q", want)
		}
	}

	wantInPublishStep := []string{
		"name: Gormes ${{ github.ref_name }} (${{ steps.release_notes.outputs.date_alias }})",
		"body_path: release-notes.md",
	}
	for _, want := range wantInPublishStep {
		if !strings.Contains(publishStep, want) {
			t.Errorf("softprops publish step missing %q", want)
		}
	}

	assertWorkflowOrder(t, notesStep,
		"date_alias=$(grep 'var VersionDateAlias =' cmd/gormes/main.go | sed 's/.*\"\\(.*\\)\".*/\\1/')",
		"echo \"# Gormes ${GITHUB_REF_NAME} (${date_alias})\"",
	)
	assertWorkflowOrder(t, notesStep,
		"echo \"# Gormes ${GITHUB_REF_NAME} (${date_alias})\"",
		"echo \"## Install\"",
	)
}

// TestReleaseWorkflowPublishesInstallScripts pins the regression
// observed during the v0.2.0 fresh-install probe: a curl following
// the natural URL pattern
//
//	https://github.com/.../releases/download/v0.2.0/install.sh
//
// hit 404. install.sh and install.ps1 were not GitHub release assets;
// users had to know a non-GitHub landing-served path to bootstrap from a
// tagged release.
//
// Contract: every tagged release MUST carry install.sh and
// install.ps1 alongside the platform tarballs. The publish step
// copies them out of the source checkout into dist/ before the
// softprops upload, and surfaces them in the release notes "Install"
// block so the URL pattern is discoverable without reading the
// landing site.
func TestReleaseWorkflowPublishesInstallScripts(t *testing.T) {
	workflow := readRepoFile(t, ".github/workflows/release.yml")
	publishStep := workflowStepBlock(t, workflow, "- uses: softprops/action-gh-release@v3")

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
	// documents the canonical curl URL and operators stay on GitHub's
	// release trust surface.
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

func TestReleasePrepGuideTargetMatrixMatchesWorkflow(t *testing.T) {
	workflow := readRepoFile(t, ".github/workflows/release.yml")
	raw, err := os.ReadFile("v0.1.0-release-prep.md")
	if err != nil {
		t.Fatalf("read release prep guide: %v", err)
	}
	guide := string(raw)

	targets := []struct {
		slug   string
		goos   string
		goarch string
	}{
		{slug: "linux-amd64", goos: "linux", goarch: "amd64"},
		{slug: "linux-arm64", goos: "linux", goarch: "arm64"},
		{slug: "darwin-amd64", goos: "darwin", goarch: "amd64"},
		{slug: "darwin-arm64", goos: "darwin", goarch: "arm64"},
		{slug: "windows-amd64", goos: "windows", goarch: "amd64"},
		{slug: "windows-arm64", goos: "windows", goarch: "arm64"},
		{slug: "android-arm64", goos: "android", goarch: "arm64"},
	}
	for _, target := range targets {
		if !strings.Contains(workflow, "goos: "+target.goos) ||
			!strings.Contains(workflow, "goarch: "+target.goarch) {
			t.Fatalf("release workflow missing target %s", target.slug)
		}
		if !strings.Contains(guide, target.slug) {
			t.Errorf("release prep guide missing target slug %s", target.slug)
		}
	}

	wantGuide := []string{
		"Release workflow still only publishes GitHub Releases from `v*` tags.",
		"Do not create or push `v0.1.0` unless",
		"SHA-256 sidecars",
		"SPDX SBOMs",
		"GitHub SBOM and build-provenance attestations",
		"android-arm64 (Termux)",
	}
	for _, want := range wantGuide {
		if !strings.Contains(guide, want) {
			t.Errorf("release prep guide missing %q", want)
		}
	}

	if strings.Contains(guide, "will build Linux, macOS, and Windows static archives") {
		t.Errorf("release prep guide still contains stale Linux/macOS/Windows summary")
	}
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
