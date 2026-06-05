package repochecks_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestBuildScriptsBacklogReadersAreSplitDirectorySafe locks the module-split
// umbrella C5e contract: the fleet shell scripts must read the canonical
// backlog through the split-safe `go run ./cmd/progress emit` (C5d) instead of
// jq-ing / require_file-ing the canonical path directly, and the two
// backlog-triggered GitHub Actions must keep firing when the canonical becomes
// a module-keyed split directory. Byte-identical monolith-vs-split output is
// already proven by progressctl's TestEmitIsSplitDirectorySafeAndCanonical;
// this test proves the non-Go consumers delegate to that proven seam and no
// longer assume a single on-disk file.
func TestBuildScriptsBacklogReadersAreSplitDirectorySafe(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))

	read := func(rel string) string {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(repoRoot, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		return string(raw)
	}

	scripts := []string{
		"scripts/gormes-architecture-planner-tasks-manager.sh",
		"scripts/documentation-improver.sh",
		"scripts/landingpage-improver.sh",
	}
	for _, rel := range scripts {
		body := read(rel)
		// No raw existence-gate of the canonical path.
		if strings.Contains(body, `require_file "$PROGRESS_JSON"`) {
			t.Errorf("%s still gates the canonical with `require_file \"$PROGRESS_JSON\"` (breaks on a split directory)", rel)
		}
		// No raw jq parse of the canonical path (the jq-program terminator
		// `' "$PROGRESS_JSON"` that fed jq the canonical file directly).
		if strings.Contains(body, `' "$PROGRESS_JSON"`) {
			t.Errorf("%s still jq-parses the canonical path directly via `' \"$PROGRESS_JSON\"` (breaks on a split directory)", rel)
		}
		// Each script must read the backlog through the split-safe seam.
		if !strings.Contains(body, "go run ./cmd/progress emit") {
			t.Errorf("%s does not read the backlog via the split-safe `go run ./cmd/progress emit`", rel)
		}
	}

	splitGlob := "architecture_plan/progress.json/**"
	for _, rel := range []string{
		".github/workflows/www-mirror-sync.yml",
		".github/workflows/deploy-gormes-www.yml",
	} {
		if !strings.Contains(read(rel), splitGlob) {
			t.Errorf("%s path-trigger does not include the split-directory glob %q; backlog changes would silently stop firing it after the C5 flip", rel, splitGlob)
		}
	}
}
