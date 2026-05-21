package piextension

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGormesLoopPromptRequiresRecentLogsSkillsAndFullCIGate(t *testing.T) {
	content := readGormesLoopExtension(t)

	mustContainAll(t, content, []string{
		"recentLoopLogSnippet",
		"Recent loop log records",
		"gormes-skill-manager",
		"gormes-delivery-loop",
		"gormes-pi-parity",
		"gormes-tdd-slice",
		"gormes-tdd",
		"gormes-git",
		"caveman",
		"go test ./... -count=1",
		"go run ./cmd/progress validate",
		"git diff --check",
		"CI_GREEN",
	})
}

func TestGormesLoopActiveDefaultDoesNotAskToReplaceState(t *testing.T) {
	content := readGormesLoopExtension(t)

	mustContainAll(t, content, []string{
		`const values = ["start", "restart", "stop", "status"]`,
		`case "restart":`,
		`startLoop(pi, ctx, parsed.topic, parsed.iterations, true)`,
		"if (state.active && !replaceActive)",
		"Use /gormes-loop restart",
	})
}

func readGormesLoopExtension(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test file")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	extensionPath := filepath.Join(repoRoot, ".pi", "extensions", "gormes-delivery-loop.ts")
	contentBytes, err := os.ReadFile(extensionPath)
	if err != nil {
		t.Fatalf("read extension: %v", err)
	}
	return string(contentBytes)
}

func mustContainAll(t *testing.T, content string, needles []string) {
	t.Helper()
	for _, needle := range needles {
		if !strings.Contains(content, needle) {
			t.Fatalf("extension prompt contract missing %q", needle)
		}
	}
}
