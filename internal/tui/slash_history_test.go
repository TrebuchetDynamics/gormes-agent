package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestHistorySlashRendersCurrentTranscriptPageWithoutSubmitting(t *testing.T) {
	sub := &nopSubmitter{}
	m := newHistorySlashModel(sub)
	longAssistant := "assistant " + strings.Repeat("x", 96)
	m.frame.History = []hermes.Message{
		{Role: "system", Content: "hidden system row"},
		{Role: "user", Content: "hello from Juan"},
		{Role: "assistant", Content: longAssistant},
		{Role: "tool", Name: "read_file", Content: "tool output stays out of /history"},
		{Role: "assistant", ToolCalls: []hermes.ToolCall{{Name: "read_file"}}},
	}

	m = enterSlashDispatchBehavior(t, m, "/history 12")

	if sub.calls != 0 {
		t.Fatalf("/history reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /history = %q, want cleared", got)
	}
	if m.transientPage == nil {
		t.Fatal("/history did not open a transient page")
	}
	if m.transientPage.Title != "History" {
		t.Fatalf("page title = %q, want History", m.transientPage.Title)
	}
	body := m.transientPage.Body
	for _, want := range []string{"[You #1]", "hello from Juan", "[Gormes #2]", "[Gormes #3]", "(1 tool calls)"} {
		if !strings.Contains(body, want) {
			t.Fatalf("history page body missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "hidden system row") || strings.Contains(body, "tool output stays out") {
		t.Fatalf("history page body included non-user/assistant rows:\n%s", body)
	}
	if strings.Contains(body, strings.Repeat("x", 96)) || !strings.Contains(body, "…") {
		t.Fatalf("history page body did not clip long assistant text:\n%s", body)
	}
	view := m.View()
	if !strings.Contains(view, "History") || !strings.Contains(view, "hello from Juan") {
		t.Fatalf("View() did not render transient history page:\n%s", view)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/history fell through to fallback: %q", m.statusMessage)
	}
}

func TestHistorySlashEmptyConversationAndDismiss(t *testing.T) {
	sub := &nopSubmitter{}
	m := newHistorySlashModel(sub)

	m = enterSlashDispatchBehavior(t, m, "/history")
	if sub.calls != 0 {
		t.Fatalf("empty /history reached Submitter %d time(s), want 0", sub.calls)
	}
	if m.transientPage != nil {
		t.Fatalf("empty /history page = %+v, want nil", *m.transientPage)
	}
	if !strings.Contains(m.statusMessage, "no conversation yet") {
		t.Fatalf("empty /history status = %q, want no conversation evidence", m.statusMessage)
	}

	m.frame.History = []hermes.Message{{Role: "user", Content: "keep visible until dismissed"}}
	m = enterSlashDispatchBehavior(t, m, "/history")
	if m.transientPage == nil {
		t.Fatal("/history with conversation did not open a transient page")
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	updated, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", next)
	}
	if updated.transientPage != nil {
		t.Fatalf("Escape left transient page open: %+v", *updated.transientPage)
	}
}

func TestHistorySlashCompletionsAndBusyAvailability(t *testing.T) {
	completions := HermesSlashCommandCompletions("/his")
	for _, completion := range completions {
		if completion.Name != "history" {
			continue
		}
		if !completion.Available {
			t.Fatalf("completion %+v marked unavailable, want available", completion)
		}
		goto foundCompletion
	}
	t.Fatalf("HermesSlashCommandCompletions(/his) = %+v, want history", completions)

foundCompletion:
	busy := NewDefaultSlashRegistry().BusyAvailableSlashes()
	for _, name := range busy {
		if name == "history" {
			return
		}
	}
	t.Fatalf("BusyAvailableSlashes() = %v, want history", busy)
}

func newHistorySlashModel(sub *nopSubmitter) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frame := kernel.RenderFrame{Phase: kernel.PhaseIdle, SessionID: "sess-history"}
	frames <- frame
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true})
	m.frame = frame
	m.width = 96
	m.height = 28
	return m
}
