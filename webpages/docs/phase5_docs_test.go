package docs_test

import (
	"os"
	"strings"
	"testing"
)

func TestPhase5DocsTrackExecuteCodeCloseout(t *testing.T) {
	phase5 := readDoc(t, "content/building-gormes/architecture_plan/phase-5-final-purge.md")
	for _, want := range []string{
		"**Status:** 🔨 in progress",
		"| 5.K — Code Execution | ✅ complete |",
		"timeout/output caps",
		"filesystem/network blocking",
	} {
		if !strings.Contains(phase5, want) {
			t.Fatalf("phase-5-final-purge.md is missing %q", want)
		}
	}

	toolExecution := readDoc(t, "content/building-gormes/core-systems/tool-execution.md")
	for _, want := range []string{
		"`execute_code`",
		"shell-only `execute_code`",
		"Python runtime",
		"filesystem/network blocking",
	} {
		if !strings.Contains(toolExecution, want) {
			t.Fatalf("tool-execution.md is missing %q", want)
		}
	}

	docsProgress, err := os.ReadFile("content/building-gormes/architecture_plan/progress.json")
	if err != nil {
		t.Fatalf("read canonical progress.json: %v", err)
	}
	for _, want := range []string{
		`"5": {`,
		`"5.K": {`,
		`"status": "complete"`,
	} {
		if !strings.Contains(string(docsProgress), want) {
			t.Fatalf("canonical progress.json is missing %q", want)
		}
	}

	siteProgress, err := os.ReadFile("../landing/legacy/go-renderer/internal/site/data/progress.json")
	if err != nil {
		t.Fatalf("read slim legacy site progress copy: %v", err)
	}
	for _, want := range []string{
		`"5.K": {`,
		`"name": "Code Execution"`,
		`"status": "complete"`,
	} {
		if !strings.Contains(string(siteProgress), want) {
			t.Fatalf("slim legacy site progress copy is missing %q", want)
		}
	}
}
