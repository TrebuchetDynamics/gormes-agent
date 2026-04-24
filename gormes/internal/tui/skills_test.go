package tui

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/skills"
)

// TestRenderSkillsPane_ContainsBrowseSummary proves the TUI surface for Phase
// 6.F renders the same deterministic browse summary that the gateway
// /skills command sends, so operators see the same view on both edges.
func TestRenderSkillsPane_ContainsBrowseSummary(t *testing.T) {
	installed := []skills.InstalledSkill{
		{Name: "alpha", Description: "alpha desc", Ref: "bundled/ops/alpha"},
	}
	available := []skills.SkillMeta{
		{Source: "clawhub", Ref: "clawhub/devops/docker-management", Name: "docker-management", Description: "Manage Docker"},
	}
	view := skills.BuildBrowseView(installed, available, 1, skills.DefaultBrowsePerPage)

	rendered := RenderSkillsPane(view, 80)
	for _, want := range []string{
		"Skills browser",
		"alpha — alpha desc",
		"docker-management — Manage Docker",
		"Page 1/1",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("RenderSkillsPane missing %q, got:\n%s", want, rendered)
		}
	}
}

// TestRenderSkillsPane_EmptyView produces the shared empty placeholders
// instead of a blank string so the TUI never renders a dead pane.
func TestRenderSkillsPane_EmptyView(t *testing.T) {
	view := skills.BuildBrowseView(nil, nil, 1, skills.DefaultBrowsePerPage)
	text := RenderSkillsPane(view, 40)
	for _, want := range []string{
		"Skills browser",
		"No installed skills",
		"No available skills",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("empty RenderSkillsPane missing %q, got:\n%s", want, text)
		}
	}
}
