package tui

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/prompttemplates"
)

type recordingSubmitter struct {
	calls int
	texts []string
}

func (r *recordingSubmitter) submit(text string) {
	r.calls++
	r.texts = append(r.texts, text)
}

func TestSkillSlashDispatch_SubmitsExpandedSkillMessage(t *testing.T) {
	sub := &recordingSubmitter{}
	m := newSkillSlashModel(sub, []skills.SkillSlashCommand{{
		Command:     "/review-skill",
		Name:        "review-skill",
		Description: "Review code",
		SkillDir:    "/tmp/review-skill",
		Skill:       skills.Skill{Name: "review-skill", Body: "Review the requested code carefully."},
	}})

	m = enterSlashDispatchBehavior(t, m, "/review-skill inspect src")

	if sub.calls != 1 {
		t.Fatalf("Submitter calls = %d, want 1", sub.calls)
	}
	got := sub.texts[0]
	for _, want := range []string{
		`[IMPORTANT: The user has invoked the "review-skill" skill`,
		"Review the requested code carefully.",
		"[Skill directory: /tmp/review-skill]",
		"The user has provided the following instruction alongside the skill invocation: inspect src",
		"[Runtime note: native-tui]",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("submitted skill message missing %q:\n%s", want, got)
		}
	}
	if strings.HasPrefix(strings.TrimSpace(got), "/review-skill") {
		t.Fatalf("raw slash leaked into submit: %q", got)
	}
	if m.editor.Value() != "" {
		t.Fatalf("editor after skill slash = %q, want cleared", m.editor.Value())
	}
	if !strings.Contains(m.statusMessage, "skill_invoked: review-skill") {
		t.Fatalf("status = %q, want skill invocation evidence", m.statusMessage)
	}
}

func TestSkillSlashDispatch_BuiltinsAndSkillsPrecedePromptTemplates(t *testing.T) {
	sub := &recordingSubmitter{}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{
		MouseTracking: true,
		SkillSlashCommands: []skills.SkillSlashCommand{
			{Command: "/help", Name: "help", Skill: skills.Skill{Name: "help", Body: "must not run"}},
			{Command: "/review-skill", Name: "review-skill", Skill: skills.Skill{Name: "review-skill", Body: "Skill body wins."}},
		},
		PromptTemplates: PromptTemplateCatalog{Templates: []prompttemplates.Template{{Name: "review-skill", Body: "Template body must not win."}}},
	})
	m.frame.Phase = kernel.PhaseIdle

	m = enterSlashDispatchBehavior(t, m, "/help")
	if sub.calls != 0 {
		t.Fatalf("/help reached Submitter %d time(s), want builtin precedence", sub.calls)
	}

	m = enterSlashDispatchBehavior(t, m, "/review-skill now")
	if sub.calls != 1 {
		t.Fatalf("/review-skill Submitter calls = %d, want skill precedence over prompt template", sub.calls)
	}
	if !strings.Contains(sub.texts[0], "Skill body wins.") || strings.Contains(sub.texts[0], "Template body must not win") {
		t.Fatalf("skill/template precedence wrong:\n%s", sub.texts[0])
	}
}

func newSkillSlashModel(sub *recordingSubmitter, commands []skills.SkillSlashCommand) Model {
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{
		MouseTracking:      true,
		SkillSlashCommands: commands,
	})
	m.frame.Phase = kernel.PhaseIdle
	return m
}
