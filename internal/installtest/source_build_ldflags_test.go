package installtest

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestInstallSourceBuildEmbedsCommitMetadata guards against silent
// regressions where install.sh's source-build path runs
// `go build -o ... ./cmd/gormes` with no `-ldflags`, leaving GitCommit and
// GitDirty at their compiled-in defaults ("unknown"/"false") so doctor
// reports `[PASS] build identity: version=… source build (no commit
// metadata)` and operators cannot tell which commit is actually running.
//
// The build invocation must carry `-ldflags` that injects at minimum
// `main.GitCommit` and `main.GitDirty` from the resolved source checkout.
func TestInstallSourceBuildEmbedsCommitMetadata(t *testing.T) {
	root := repoRoot(t)
	body, err := os.ReadFile(filepath.Join(root, "install.sh"))
	if err != nil {
		t.Fatalf("read install.sh: %v", err)
	}
	src := string(body)

	// Find the `go build ... ./cmd/gormes` invocation that produces the
	// managed binary. We accept any whitespace/escaping but the line must
	// include both `-ldflags` and references to the version vars.
	re := regexp.MustCompile(`go build[^\n]*\./cmd/gormes`)
	matches := re.FindAllString(src, -1)
	if len(matches) == 0 {
		t.Fatal("install.sh: no `go build ./cmd/gormes` invocation found")
	}

	for _, line := range matches {
		if !strings.Contains(line, "-ldflags") {
			t.Errorf("source-build go build line missing -ldflags:\n  %s", line)
			continue
		}
		// The flags string is built from a shell variable; just require the
		// invocation references it (e.g. $LDFLAGS or "$ldflags").
		if !strings.Contains(line, "main.GitCommit") &&
			!strings.Contains(line, "ldflags") {
			t.Errorf("source-build go build line does not appear to inject GitCommit:\n  %s", line)
		}
	}

	// Stronger assertion: somewhere in install.sh, the ldflags string must
	// reference both main.GitCommit and main.GitDirty so doctor sees the
	// real commit + dirty state instead of compiled-in defaults.
	for _, want := range []string{"main.GitCommit", "main.GitDirty"} {
		if !strings.Contains(src, want) {
			t.Errorf("install.sh does not inject %s via -ldflags", want)
		}
	}
}
