package wizard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestWizardSearchPickRenderClearsAndClampsFilteredRows(t *testing.T) {
	step := Pick("provider", "Provider", []Choice{
		{ID: "opencode", Label: "OpenCode Zen (35+ curated models, premium subscription)"},
		{ID: "openrouter", Label: "OpenRouter (100+ models, pay-per-use)"},
		{ID: "openai", Label: "OpenAI Codex"},
		{ID: "huggingface", Label: "Hugging Face Inference Providers (20+ open models)"},
	}, WithSearchChoices())
	m := newModel([]Step{step})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 48, Height: 12})
	m = updated.(model)
	for _, r := range "opena" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(model)
	}

	got := m.View()
	if strings.Count(got, "Filter:") != 1 {
		t.Fatalf("search pick rendered stale/duplicate filter rows:\n%s", got)
	}
	for _, line := range strings.Split(got, "\n") {
		if width := lipgloss.Width(line); width > 48 {
			t.Fatalf("search pick line width %d exceeds 48:\n%q\n\n%s", width, line, got)
		}
	}
}
