package codingagents

import (
	"strings"
	"testing"
)

func TestWrapPrompt_IncludesCoreFields(t *testing.T) {
	t.Parallel()
	req := CodingAgentRequest{
		Workspace:  "/tmp/some/project",
		Prompt:     "refactor handler",
		Mode:       ModeEdit,
		AllowEdits: true,
	}
	out := WrapPrompt(req)
	for _, want := range []string{
		"You are being run by gormes as a coding worker.",
		"Workspace: /tmp/some/project",
		"Mode: edit",
		"Edits allowed: true",
		"refactor handler",
		"Stay inside the Workspace",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("WrapPrompt missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Gormes Agent") {
		t.Fatalf("WrapPrompt must not emit deprecated Gormes Agent wording:\n%s", out)
	}
	if strings.Contains(out, "Gormes-repo rules:") {
		t.Fatalf("non-gormes workspace should not include repo rules:\n%s", out)
	}
}

func TestWrapPrompt_InjectsGormesRepoRules(t *testing.T) {
	t.Parallel()
	req := CodingAgentRequest{
		Workspace:  "/home/dev/projects/gormes-agent",
		Prompt:     "fix progress validate",
		Mode:       ModePlan,
		AllowEdits: false,
	}
	out := WrapPrompt(req)
	if !strings.Contains(out, "Gormes-repo rules:") {
		t.Fatalf("expected Gormes-repo rules block, got:\n%s", out)
	}
	if !strings.Contains(out, "development branch") {
		t.Fatalf("expected development-branch rule, got:\n%s", out)
	}
}

func TestWrapPrompt_DefaultModeWhenEmpty(t *testing.T) {
	t.Parallel()
	out := WrapPrompt(CodingAgentRequest{Workspace: "/tmp/x", Prompt: "ask"})
	if !strings.Contains(out, "Mode: explain") {
		t.Fatalf("expected default Mode: explain, got:\n%s", out)
	}
}
