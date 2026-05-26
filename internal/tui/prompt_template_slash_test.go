package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/prompttemplates"
)

func TestPromptTemplateSlashExpansionSeedsEditorWithoutSubmit(t *testing.T) {
	sub := &recordingPromptTemplateSubmitter{}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	catalog := prompttemplates.Catalog{Templates: []prompttemplates.Template{
		{Name: "review", Description: "Review staged changes", ArgumentHint: "<scope>", Body: "Review $1 with args: $ARGUMENTS"},
	}}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{PromptTemplates: catalog})
	m.frame.Phase = kernel.PhaseIdle

	m = enterSlashDispatchBehavior(t, m, `/review staged "bug fix"`)

	if sub.calls != 0 {
		t.Fatalf("/review reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "Review staged with args: staged bug fix" {
		t.Fatalf("editor after template expansion = %q", got)
	}
	if !strings.Contains(m.statusMessage, "prompt_template_expanded") || !strings.Contains(m.statusMessage, "review") {
		t.Fatalf("status after template expansion = %q", m.statusMessage)
	}

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		_ = cmd()
	}
	updated, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", next)
	}
	if updated.editor.Value() != "" {
		t.Fatalf("editor after submitting expanded template = %q, want cleared", updated.editor.Value())
	}
	if sub.calls != 1 || sub.last != "Review staged with args: staged bug fix" {
		t.Fatalf("submitter calls=%d last=%q, want one expanded prompt", sub.calls, sub.last)
	}
}

type recordingPromptTemplateSubmitter struct {
	calls int
	last  string
}

func (s *recordingPromptTemplateSubmitter) submit(text string) {
	s.calls++
	s.last = text
}

func TestPromptTemplateSlashCompletions(t *testing.T) {
	catalog := prompttemplates.Catalog{Templates: []prompttemplates.Template{
		{Name: "review", Description: "Review staged changes", ArgumentHint: "<scope>"},
	}}
	completions := SlashCompletionsWithPromptTemplates("/rev", catalog)
	if len(completions) != 1 {
		t.Fatalf("SlashCompletionsWithPromptTemplates = %+v, want one template", completions)
	}
	if got := completions[0]; got.Name != "review" || got.ArgumentHint != "<scope>" || got.Description != "Review staged changes" {
		t.Fatalf("template completion = %+v", got)
	}
	menu := renderSlashCompletionMenuWithTemplates("/rev", 80, DefaultHermesSkin(), catalog)
	if !strings.Contains(menu, "/review <scope>") || !strings.Contains(menu, "Review staged changes") {
		t.Fatalf("completion menu missing prompt template hint/description:\n%s", menu)
	}

	// Built-in slash commands keep precedence over prompt-template names.
	colliding := prompttemplates.Catalog{Templates: []prompttemplates.Template{{Name: "skills", Body: "shadow"}}}
	sub := &nopSubmitter{}
	calls := 0
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{
		PromptTemplates: colliding,
		SkillsCommand: func(input string) string {
			calls++
			return "skills local command"
		},
	})
	m = enterSlashDispatchBehavior(t, m, "/skills list")
	if calls != 1 {
		t.Fatalf("colliding /skills template shadowed built-in skills command; calls=%d", calls)
	}
	if sub.calls != 0 {
		t.Fatalf("/skills list reached Submitter %d time(s), want 0", sub.calls)
	}
}
