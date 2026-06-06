package installtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/workspace"
)

func TestInstallersRequireCurrentGoToolchain(t *testing.T) {
	root := repoRoot(t)
	goMod := readFileFromRoot(t, root, "go.mod")
	if !strings.Contains(goMod, "\ngo 1.26\n") {
		t.Fatalf("go.mod must declare Go 1.26; got:\n%s", goMod)
	}
	if !strings.Contains(goMod, "\ntoolchain go1.26.") {
		t.Fatalf("go.mod must declare a Go 1.26 toolchain; got:\n%s", goMod)
	}

	installSH := readFileFromRoot(t, root, "install.sh")
	wantInstallSH := []string{
		`GO_VERSION="${GORMES_GO_VERSION:-1.26.0}"`,
		"Go 1.26+ required",
		"go1.2[6-9]*|go1.[3-9][0-9]*|go[2-9]*)",
	}
	for _, want := range wantInstallSH {
		if !strings.Contains(installSH, want) {
			t.Errorf("install.sh missing %q", want)
		}
	}
	if strings.Contains(installSH, "Go 1.25+ required") ||
		strings.Contains(installSH, `GO_VERSION="${GORMES_GO_VERSION:-1.25.0}"`) ||
		strings.Contains(installSH, "go1.2[5-9]*") {
		t.Error("install.sh still accepts or advertises stale Go 1.25")
	}

	installPS1 := readFileFromRoot(t, root, "scripts/install.ps1")
	wantInstallPS1 := []string{
		"GORMES_GO_VERSION    managed Go fallback version (default: 1.26.0)",
		"} else { '1.26.0' }",
		"Go 1.26+ required",
		"^go1\\.(2[6-9]|[3-9][0-9])",
	}
	for _, want := range wantInstallPS1 {
		if !strings.Contains(installPS1, want) {
			t.Errorf("scripts/install.ps1 missing %q", want)
		}
	}
	if strings.Contains(installPS1, "Go 1.25+ required") ||
		strings.Contains(installPS1, "default: 1.25.0") ||
		strings.Contains(installPS1, "'1.25.0'") ||
		strings.Contains(installPS1, "2[5-9]") {
		t.Error("scripts/install.ps1 still accepts or advertises stale Go 1.25")
	}

	for _, rel := range []string{
		"webpages/docs/content/install/from-source.md",
		"webpages/docs/content/start-here/_index.md",
	} {
		body := readFileFromRoot(t, root, rel)
		if !strings.Contains(body, "Go 1.26+") && !strings.Contains(body, "default `1.26.0`") {
			t.Errorf("%s must advertise Go 1.26+ or managed Go 1.26.0 for source builds", rel)
		}
		if strings.Contains(body, "Go 1.25+") {
			t.Errorf("%s still advertises stale Go 1.25+", rel)
		}
		if strings.Contains(body, "default `1.25.0`") {
			t.Errorf("%s still advertises stale managed Go default 1.25.0", rel)
		}
	}
}

func readFileFromRoot(t *testing.T, root, rel string) string {
	t.Helper()
	path := filepath.Join(root, rel)
	if filepath.ToSlash(rel) == "webpages/docs/content/building-gormes/architecture_plan/progress.json" {
		return readLogicalProgressFromRoot(t, path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(raw)
}

func readLogicalProgressFromRoot(t *testing.T, path string) string {
	t.Helper()
	ws := workspace.New(filepath.Clean(filepath.Join(path, "..", "..", "..", "..", "..", "..")))
	raw, err := ws.EmitBytes()
	if err != nil {
		t.Fatalf("workspace.EmitBytes(%s): %v", path, err)
	}
	return string(raw)
}
