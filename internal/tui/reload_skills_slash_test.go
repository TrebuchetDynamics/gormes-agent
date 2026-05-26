package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
)

func TestReloadSkillsSlashRefreshesSkillSlashRegistry(t *testing.T) {
	sub := &recordingSubmitter{}
	calls := 0
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{
		MouseTracking: true,
		SkillSlashReload: func(context.Context) (SkillSlashReloadResult, error) {
			calls++
			return SkillSlashReloadResult{Commands: []skills.SkillSlashCommand{{
				Command:     "/fresh-skill",
				Name:        "fresh-skill",
				Description: "Fresh skill",
				Skill:       skills.Skill{Name: "fresh-skill", Body: "Fresh skill body."},
			}}, Output: "Skills Reloaded\n1 skill(s) available"}, nil
		},
	})
	m.frame.Phase = kernel.PhaseIdle

	m = enterSlashDispatchBehavior(t, m, "/reload-skills")
	if calls != 1 {
		t.Fatalf("reload calls = %d, want 1", calls)
	}
	if sub.calls != 0 {
		t.Fatalf("/reload-skills reached Submitter %d time(s), want 0", sub.calls)
	}
	if !strings.Contains(m.statusMessage, "1 skill(s) available") {
		t.Fatalf("status after reload = %q", m.statusMessage)
	}

	m = enterSlashDispatchBehavior(t, m, "/fresh-skill now")
	if sub.calls != 1 {
		t.Fatalf("/fresh-skill submit calls = %d, want 1", sub.calls)
	}
	if !strings.Contains(sub.texts[0], "Fresh skill body.") || strings.Contains(sub.texts[0], "/fresh-skill now") {
		t.Fatalf("fresh skill submit did not expand correctly:\n%s", sub.texts[0])
	}
}

func TestReloadSkillsSlashConsumesUnavailableAndFailure(t *testing.T) {
	sub := &recordingSubmitter{}
	m := newSkillSlashModel(sub, nil)

	m = enterSlashDispatchBehavior(t, m, "/reload_skills")
	if sub.calls != 0 {
		t.Fatalf("unwired reload reached Submitter %d time(s), want 0", sub.calls)
	}
	if !strings.Contains(m.statusMessage, "reload-skills") || !strings.Contains(m.statusMessage, "unavailable") {
		t.Fatalf("unwired reload status = %q, want unavailable evidence", m.statusMessage)
	}

	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	m = NewModelWithOptions(frames, sub.submit, func() {}, Options{
		MouseTracking: true,
		SkillSlashReload: func(context.Context) (SkillSlashReloadResult, error) {
			return SkillSlashReloadResult{}, errors.New("scan failed")
		},
	})
	m.frame.Phase = kernel.PhaseIdle
	m = enterSlashDispatchBehavior(t, m, "/reload-skills")
	if !strings.Contains(m.statusMessage, "scan failed") {
		t.Fatalf("failed reload status = %q, want error evidence", m.statusMessage)
	}
}
