package repoctl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMakefileUsesFocusedProgressAndRepoHelpers(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRootForTest(t), "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	makefile := string(raw)

	for _, forbidden := range []string{
		"bash scripts/record-benchmark.sh",
		"bash scripts/record-progress.sh",
		"bash scripts/update-readme.sh",
		"go run ./cmd/progress-gen",
		"go run ./cmd/builder-loop",
		"go run ./cmd/planner-loop",
	} {
		if strings.Contains(makefile, forbidden) {
			t.Fatalf("Makefile still calls %q", forbidden)
		}
	}

	for _, required := range []string{
		"go run ./cmd/gormes-repo benchmark record",
		"go run ./cmd/progress write",
		"go run ./cmd/gormes-repo readme update",
	} {
		if !strings.Contains(makefile, required) {
			t.Fatalf("Makefile missing %q", required)
		}
	}
}

func TestMakefileVersionExtractorMatchesGoVarDeclaration(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRootForTest(t), "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	makefile := string(raw)
	if !strings.Contains(makefile, "(var[[:space:]]+)?Version") {
		t.Fatalf("Makefile VERSION extractor must match `var Version = ...`; got:\n%s", makefile)
	}
}

func TestLegacyGoRendererMakefileUsesSplitSafeProgressGeneration(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRootForTest(t), "webpages", "landing", "legacy", "go-renderer", "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	makefile := string(raw)

	for _, forbidden := range []string{
		"PROGRESS_SRC :=",
		"cp $(PROGRESS_SRC)",
		"architecture_plan/progress.json",
	} {
		if strings.Contains(makefile, forbidden) {
			t.Fatalf("legacy go-renderer Makefile must not raw-copy canonical progress path %q; got:\n%s", forbidden, makefile)
		}
	}

	for _, required := range []string{
		".PHONY: build run test fmt clean progress-data",
		"build: $(DATA_DIR)/benchmarks.json progress-data",
		"cd ../../../.. && go run ./cmd/progress write",
	} {
		if !strings.Contains(makefile, required) {
			t.Fatalf("legacy go-renderer Makefile missing split-safe progress generation %q; got:\n%s", required, makefile)
		}
	}
}
