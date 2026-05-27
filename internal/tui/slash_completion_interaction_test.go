package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/prompttemplates"
)

func TestSlashCompletionInteraction_UpDownOwnMenuBeforeHistory(t *testing.T) {
	m := NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, Options{OfflineSmoke: true})
	m.frame.Phase = kernel.PhaseIdle
	m.width = 80
	m.height = 20

	m.editor.SetValue("older prompt")
	m = updateModelCompletionKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	m.editor.SetValue("/r")

	m = updateModelCompletionKey(t, m, tea.KeyMsg{Type: tea.KeyUp})
	if got := m.editor.Value(); got != "/r" {
		t.Fatalf("Up with slash completions active changed editor to %q, want draft preserved and history not recalled", got)
	}

	m = updateModelCompletionKey(t, m, tea.KeyMsg{Type: tea.KeyDown})
	m = updateModelCompletionKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if got := m.editor.Value(); got == "/r" || got == "older prompt" || !strings.HasPrefix(got, "/") {
		t.Fatalf("Enter after completion navigation editor = %q, want accepted slash completion, not history", got)
	}
}

func TestSlashCompletionInteraction_EnterAcceptsNonExactBeforeDispatch(t *testing.T) {
	sub := &nopSubmitter{}
	m := NewModelWithOptions(make(chan kernel.RenderFrame), sub.submit, func() {}, Options{OfflineSmoke: true})
	m.frame.Phase = kernel.PhaseIdle
	m.width = 80
	m.height = 20
	m.editor.SetValue("/he")

	m = updateModelCompletionKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if got := m.editor.Value(); got != "/help" {
		t.Fatalf("Enter on non-exact slash completion editor = %q, want /help", got)
	}
	if sub.calls != 0 {
		t.Fatalf("Enter on non-exact completion reached submitter %d times, want 0", sub.calls)
	}
	if m.transientPage != nil {
		t.Fatalf("Enter on non-exact completion opened page %+v, want accept-only", *m.transientPage)
	}

	m = updateModelCompletionKey(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if got := m.editor.Value(); got != "" {
		t.Fatalf("second Enter after exact /help editor = %q, want dispatched and cleared", got)
	}
	if !strings.Contains(m.statusMessage, "Native TUI commands") {
		t.Fatalf("second Enter after exact /help status = %q, want dispatched slash help", m.statusMessage)
	}
}

func TestSlashCompletionInteraction_TabAddsArgumentSpaceAndNoPlaceholder(t *testing.T) {
	m := NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, Options{
		PromptTemplates: prompttemplates.Catalog{Templates: []prompttemplates.Template{{
			Name:         "zz-review",
			Description:  "review a scope",
			ArgumentHint: "<scope>",
			Body:         "review ${@:1}",
		}}},
	})
	m.frame.Phase = kernel.PhaseIdle
	m.width = 80
	m.height = 20
	m.editor.SetValue("/zz")

	m = updateModelCompletionKey(t, m, tea.KeyMsg{Type: tea.KeyTab})
	if got := m.editor.Value(); got != "/zz-review " {
		t.Fatalf("Tab accepted template completion = %q, want command token plus trailing space only", got)
	}
	if strings.Contains(m.editor.Value(), "<scope>") {
		t.Fatalf("Tab inserted placeholder text into editor: %q", m.editor.Value())
	}
}

func TestSlashCompletionInteraction_EscapeDismissesMenuKeepsDraft(t *testing.T) {
	m := NewModelWithOptions(make(chan kernel.RenderFrame), func(string) {}, func() {}, Options{})
	m.frame.Phase = kernel.PhaseIdle
	m.width = 80
	m.height = 20
	m.editor.SetValue("/he")

	if view := m.View(); !strings.Contains(view, "Search /he") {
		t.Fatalf("precondition: slash completion menu not visible:\n%s", view)
	}

	m = updateModelCompletionKey(t, m, tea.KeyMsg{Type: tea.KeyEscape})
	if got := m.editor.Value(); got != "/he" {
		t.Fatalf("Escape changed draft to %q, want /he", got)
	}
	if view := m.View(); strings.Contains(view, "Search /he") {
		t.Fatalf("Escape did not dismiss slash completion menu:\n%s", view)
	}
}

func updateModelCompletionKey(t *testing.T, m Model, msg tea.KeyMsg) Model {
	t.Helper()
	next, cmd := m.Update(msg)
	runTestCmd(t, cmd)
	updated, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", next)
	}
	return updated
}
