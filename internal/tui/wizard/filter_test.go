package wizard

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestFilterChoices_EmptyQueryReturnsAll(t *testing.T) {
	choices := []Choice{
		{ID: "qwen/qwen-2.5-72b-instruct", Label: "Qwen2.5 72B Instruct"},
		{ID: "qwen/qwen-2.5-7b-instruct", Label: "Qwen2.5 7B Instruct"},
		{ID: "anthropic/claude-opus-4", Label: "Claude Opus 4"},
	}
	filtered, indices := FilterChoices(choices, "")
	if len(filtered) != 3 || len(indices) != 3 {
		t.Fatalf("empty query should return all choices; got %d filtered, %d indices", len(filtered), len(indices))
	}
	for i, idx := range indices {
		if idx != i {
			t.Fatalf("indices[%d] = %d, want %d", i, idx, i)
		}
	}
}

func TestFilterChoices_SubstringMatchByID(t *testing.T) {
	choices := []Choice{
		{ID: "qwen/qwen-2.5-72b-instruct", Label: "Qwen2.5 72B Instruct"},
		{ID: "qwen/qwen-2.5-7b-instruct", Label: "Qwen2.5 7B Instruct"},
		{ID: "anthropic/claude-opus-4", Label: "Claude Opus 4"},
	}
	filtered, indices := FilterChoices(choices, "qwen")
	if len(filtered) != 2 {
		t.Fatalf("query 'qwen' should match 2 choices; got %d: %#v", len(filtered), filtered)
	}
	for _, idx := range indices {
		if !strings.Contains(choices[idx].ID, "qwen") {
			t.Fatalf("index %d points to non-qwen choice: %#v", idx, choices[idx])
		}
	}
}

func TestFilterChoices_SubstringMatchByLabel(t *testing.T) {
	choices := []Choice{
		{ID: "qwen/qwen-2.5-72b-instruct", Label: "Qwen2.5 72B Instruct"},
		{ID: "anthropic/claude-opus-4", Label: "Claude Opus 4"},
	}
	filtered, _ := FilterChoices(choices, "opus")
	if len(filtered) != 1 || filtered[0].ID != "anthropic/claude-opus-4" {
		t.Fatalf("query 'opus' should match Claude Opus 4; got %#v", filtered)
	}
}

func TestFilterChoices_MultiTokenAND(t *testing.T) {
	choices := []Choice{
		{ID: "qwen/qwen-2.5-72b-instruct", Label: "Qwen2.5 72B Instruct"},
		{ID: "qwen/qwen-2.5-7b-instruct", Label: "Qwen2.5 7B Instruct"},
		{ID: "qwen/qwen3-235b-a22b", Label: "Qwen3 235B"},
	}
	filtered, _ := FilterChoices(choices, "qwen 72b")
	if len(filtered) != 1 || filtered[0].ID != "qwen/qwen-2.5-72b-instruct" {
		t.Fatalf("query 'qwen 72b' should match only 72b model; got %#v", filtered)
	}
}

func TestFilterChoices_CaseInsensitive(t *testing.T) {
	choices := []Choice{
		{ID: "anthropic/claude-opus-4", Label: "Claude Opus 4"},
	}
	filtered, _ := FilterChoices(choices, "CLAUDE")
	if len(filtered) != 1 {
		t.Fatalf("query 'CLAUDE' should match case-insensitively; got %d", len(filtered))
	}
}

func TestFilterChoices_NoMatchesReturnsEmpty(t *testing.T) {
	choices := []Choice{
		{ID: "anthropic/claude-opus-4", Label: "Claude Opus 4"},
	}
	filtered, indices := FilterChoices(choices, "zzzzzzz")
	if len(filtered) != 0 || len(indices) != 0 {
		t.Fatalf("non-matching query should return empty; got %d filtered, %d indices", len(filtered), len(indices))
	}
}

func TestSearchPick_RenderShowsFilterAndChoices(t *testing.T) {
	m := newModel([]Step{
		Pick("model", "Select model", []Choice{
			{ID: "qwen/qwen-2.5-72b-instruct", Label: "Qwen2.5 72B Instruct"},
			{ID: "qwen/qwen-2.5-7b-instruct", Label: "Qwen2.5 7B Instruct"},
			{ID: "anthropic/claude-opus-4", Label: "Claude Opus 4"},
		}, WithSearchChoices()),
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(model)

	got := m.View()
	for _, want := range []string{"Filter", "Qwen2.5", "Claude Opus", "Type to filter", "Enter select"} {
		if !strings.Contains(got, want) {
			t.Fatalf("search pick view missing %q:\n%s", want, got)
		}
	}
}

func TestSearchPick_FilteredRowsClearStaleTerminalTails(t *testing.T) {
	m := newModel([]Step{
		Pick("provider", "Select provider", []Choice{
			{ID: "openrouter", Label: "OpenRouter (100+ models, pay-per-use)"},
			{ID: "anthropic", Label: "Anthropic (Claude models - API key or Claude Code)"},
			{ID: "openai-codex", Label: "OpenAI Codex"},
		}, WithSearchChoices()),
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 16})
	m = updated.(model)
	for _, r := range "codex" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(model)
	}

	got := m.View()
	for _, forbidden := range []string{"Codex" + "laude", "Codex" + "100+", "OpenAI Codex" + "laude", "OpenAI Codex" + "100+"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("filtered provider view has stale concatenated tail %q:\n%s", forbidden, got)
		}
	}
	var codexRow string
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "OpenAI Codex") {
			codexRow = line
			break
		}
	}
	if codexRow == "" {
		t.Fatalf("filtered provider view missing OpenAI Codex row:\n%s", got)
	}
	if !strings.Contains(codexRow, "\x1b[K") {
		t.Fatalf("filtered choice row %q does not clear stale terminal tails", codexRow)
	}
}

func TestSearchPick_TypingFiltersChoices(t *testing.T) {
	m := newModel([]Step{
		Pick("model", "Select model", []Choice{
			{ID: "qwen/qwen-2.5-72b-instruct", Label: "Qwen2.5 72B Instruct"},
			{ID: "qwen/qwen-2.5-7b-instruct", Label: "Qwen2.5 7B Instruct"},
			{ID: "anthropic/claude-opus-4", Label: "Claude Opus 4"},
		}, WithSearchChoices()),
	})
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	// Type "qwen" — should filter to 2 choices
	tm.Type("qwen")

	// Enter selects the first filtered match (a qwen model)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	final := tm.FinalModel(t).(model)
	if final.err != nil {
		t.Fatalf("wizard error: %v", final.err)
	}
	choice := final.result.Choice("model")
	if !strings.HasPrefix(choice, "qwen/") {
		t.Fatalf("expected a qwen model selection after typing 'qwen', got %q", choice)
	}
}

func TestSearchPick_ArrowKeyNavigation(t *testing.T) {
	m := newModel([]Step{
		Pick("model", "Select model", []Choice{
			{ID: "model-a", Label: "Model A"},
			{ID: "model-b", Label: "Model B"},
			{ID: "model-c", Label: "Model C"},
		}, WithSearchChoices()),
	})

	// Arrow down should move cursor in filtered list
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(model)
	if m.pickCursor != 1 {
		t.Fatalf("after Down, pickCursor = %d, want 1", m.pickCursor)
	}

	// Arrow down again
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(model)
	if m.pickCursor != 2 {
		t.Fatalf("after Down x2, pickCursor = %d, want 2", m.pickCursor)
	}

	// Arrow down at end — stay
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(model)
	if m.pickCursor != 2 {
		t.Fatalf("after Down at end, pickCursor = %d, want 2", m.pickCursor)
	}

	// Arrow up
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(model)
	if m.pickCursor != 1 {
		t.Fatalf("after Up, pickCursor = %d, want 1", m.pickCursor)
	}
}

func TestSearchPick_EnterConfirmsSelection(t *testing.T) {
	m := newModel([]Step{
		Pick("model", "Select model", []Choice{
			{ID: "model-a", Label: "Model A"},
			{ID: "model-b", Label: "Model B"},
		}, WithSearchChoices()),
	})

	// Move to second item
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(model)

	// Enter to confirm
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)
	if cmd == nil {
		t.Fatal("Enter should return a quit command")
	}

	if !m.done {
		t.Fatal("model should be done after Enter")
	}
	if got := m.result.Choice("model"); got != "model-b" {
		t.Fatalf("selected model = %q, want model-b", got)
	}
}

func TestSearchPick_FilterAndSelectFromNarrowedList(t *testing.T) {
	m := newModel([]Step{
		Pick("model", "Select model", []Choice{
			{ID: "model-a", Label: "Model A"},
			{ID: "model-b", Label: "Model B"},
			{ID: "model-c", Label: "Model C"},
		}, WithSearchChoices()),
	})

	// Type "b" to filter to model-b only
	ti, _ := m.searchInput.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m.searchInput = ti
	m.pickFiltered, m.pickFilteredIndices = FilterChoices(m.steps[0].Choices, "b")
	m.pickCursor = 0

	// Enter should select the original model-b (index 1 in choices)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(model)

	if got := m.result.Choice("model"); got != "model-b" {
		t.Fatalf("after filter+enter, selected = %q, want model-b", got)
	}
}

func TestSearchPick_FilterResetsCursorIfOutOfRange(t *testing.T) {
	m := newModel([]Step{
		Pick("model", "Select model", []Choice{
			{ID: "model-a", Label: "Model A"},
			{ID: "model-b", Label: "Model B"},
			{ID: "model-c", Label: "Model C"},
		}, WithSearchChoices()),
	})

	// Move cursor to last item, then type a filter that narrows to one item.
	m.pickCursor = 2
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(model)

	if m.pickCursor != 0 {
		t.Fatalf("cursor should reset to 0 when filter narrows; got %d (filtered len=%d)", m.pickCursor, len(m.pickFiltered))
	}
	if len(m.pickFiltered) != 1 || m.pickFiltered[0].ID != "model-a" {
		t.Fatalf("filter should narrow to model-a; got %#v", m.pickFiltered)
	}
}

func TestSearchPick_ViewHardeningWideAndNarrow(t *testing.T) {
	for _, size := range []struct{ width, height int }{{20, 12}, {40, 12}, {80, 24}} {
		t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
			m := newModel([]Step{
				Pick("model", "Select model", []Choice{
					{ID: "qwen/qwen-2.5-72b-instruct", Label: "Qwen2.5 72B Instruct"},
					{ID: "anthropic/claude-opus-4", Label: "Claude Opus 4"},
				}, WithSearchChoices()),
			})
			updated, _ := m.Update(tea.WindowSizeMsg{Width: size.width, Height: size.height})
			m = updated.(model)
			got := m.View()
			if strings.TrimSpace(got) == "" {
				t.Fatal("search pick View returned blank output")
			}
			for _, line := range strings.Split(got, "\n") {
				if w := lipgloss.Width(line); w > size.width {
					t.Fatalf("line width %d exceeds terminal width %d:\n%q\n\nfull output:\n%s", w, size.width, line, got)
				}
			}
		})
	}
}

func TestSearchPick_CtrlNCtrlPNavigate(t *testing.T) {
	m := newModel([]Step{
		Pick("model", "Select model", []Choice{
			{ID: "model-a", Label: "Model A"},
			{ID: "model-b", Label: "Model B"},
		}, WithSearchChoices()),
	})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	m = updated.(model)
	if m.pickCursor != 1 {
		t.Fatalf("after ctrl+n, pickCursor = %d, want 1", m.pickCursor)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	m = updated.(model)
	if m.pickCursor != 0 {
		t.Fatalf("after ctrl+p, pickCursor = %d, want 0", m.pickCursor)
	}
}
