package docs_test

import (
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

	// Canonical backlog is read via internal/progress.Load (canonicalProgressBytes)
	// so this gate stays green whether the canonical path is a monolithic file
	// or a module-keyed split directory (module-split umbrella C5/C5c).
	docsProgress := canonicalProgressBytes(t, canonicalProgressPath)
	for _, want := range []string{
		`"5": {`,
		`"5.K": {`,
		`"status": "complete"`,
	} {
		if !strings.Contains(string(docsProgress), want) {
			t.Fatalf("canonical progress.json is missing %q", want)
		}
	}
}
