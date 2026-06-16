package tuiapp

import (
	"reflect"
	"slices"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestWelcomeToolsetsClassifiesHermesVisibleToolsets(t *testing.T) {
	got := welcomeToolsets([]llm.ToolDescriptor{
		{Name: "browser_click"},
		{Name: "browser_dialog"},
		{Name: "computer_use"},
		{Name: "discord"},
		{Name: "himalaya"},
	})
	for _, want := range []string{"browser", "browser-cdp", "computer_use", "discord", "email"} {
		if !slices.Contains(got, want) {
			t.Fatalf("welcomeToolsets() = %v, want %q for Hermes-visible startup toolset rows", got, want)
		}
	}
}

func TestWelcomeSkillRowsGroupsCategoriesWithHermesStylePreviews(t *testing.T) {
	rows := []skills.SkillRow{
		{Category: "creative", Name: "ascii-video"},
		{Category: "creative", Name: "ascii-art"},
		{Category: "devops", Name: "kanban-worker"},
		{Category: "devops", Name: "kanban-orchestrator"},
		{Category: "research", Name: "zeta"},
		{Category: "research", Name: "alpha"},
		{Category: "research", Name: "beta"},
		{Category: "research", Name: "gamma"},
		{Category: "research", Name: "delta"},
	}

	got := welcomeSkillRows(rows)
	want := []string{
		"creative: ascii-art, ascii-video",
		"devops: kanban-orchestrator, kanban-worker",
		"research: alpha, beta, delta, gamma, ...",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("welcomeSkillRows() = %#v, want %#v", got, want)
	}
}
