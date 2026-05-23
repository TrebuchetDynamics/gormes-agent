package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestSessionsSlashOpensPickerPageWithoutSubmitting(t *testing.T) {
	sub := &nopSubmitter{}
	var limits []int
	m := newSessionsSlashModel(sub, func(limit int) ([]SessionDirectoryEntry, error) {
		limits = append(limits, limit)
		return []SessionDirectoryEntry{
			{ID: "sess-new", Title: "New Work", Preview: "ask gormes to continue", Source: "cli", LastActiveAt: 200, MessageCount: 3},
			{ID: "sess-old", Title: "", Preview: "older preview", Source: "telegram", LastActiveAt: 100, MessageCount: 1},
		}, nil
	})

	m = enterSlashDispatchBehavior(t, m, "/resume")
	if sub.calls != 0 {
		t.Fatalf("/resume reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /resume = %q, want cleared", got)
	}
	if len(limits) != 1 || limits[0] != 20 {
		t.Fatalf("/resume limits = %v, want default 20", limits)
	}
	if !strings.Contains(m.statusMessage, "sessions opened") {
		t.Fatalf("/resume status = %q, want sessions opened", m.statusMessage)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/resume fell through to fallback: %q", m.statusMessage)
	}
	assertSessionsPageContains(t, m, "New Work", "ask gormes to continue", "sess-new", "3 messages", "older preview", "telegram")

	m = enterSlashDispatchBehavior(t, m, "/sessions 1")
	if len(limits) != 2 || limits[1] != 1 {
		t.Fatalf("/sessions 1 limits = %v, want second call limit 1", limits)
	}
	assertSessionsPageContains(t, m, "New Work", "sess-new")
}

func TestResumeSlashWithSessionIDSwitchesVisibleSessionAndHistoryWithoutSubmitting(t *testing.T) {
	sub := &nopSubmitter{}
	var requested string
	m := newSessionsSlashModelWithOptions(sub, nil, func(ctx context.Context, query string) (SessionResumeResult, error) {
		if ctx == nil {
			t.Fatal("resume context is nil")
		}
		requested = query
		return SessionResumeResult{
			SessionID: "sess-target",
			History: []hermes.Message{
				{Role: "user", Content: "previous question"},
				{Role: "assistant", Content: "previous answer"},
			},
		}, nil
	})
	m.transientPage = &TransientPageState{Title: "Sessions", Body: "old picker"}

	m = enterSlashDispatchBehavior(t, m, "/resume sess-tar")
	if sub.calls != 0 {
		t.Fatalf("/resume <id> reached Submitter %d time(s), want 0", sub.calls)
	}
	if requested != "sess-tar" {
		t.Fatalf("resume query = %q, want sess-tar", requested)
	}
	if got := m.SessionID(); got != "sess-target" {
		t.Fatalf("SessionID() = %q, want sess-target", got)
	}
	if got := m.frame.SessionID; got != "sess-target" {
		t.Fatalf("frame.SessionID = %q, want sess-target", got)
	}
	if len(m.frame.History) != 2 || m.frame.History[0].Content != "previous question" || m.frame.History[1].Content != "previous answer" {
		t.Fatalf("frame.History = %+v, want replayed transcript", m.frame.History)
	}
	if m.transientPage != nil {
		t.Fatalf("transientPage after resume = %+v, want nil", *m.transientPage)
	}
	if !strings.Contains(m.statusMessage, "resumed sess-target (2 messages)") {
		t.Fatalf("resume status = %q, want resumed sess-target", m.statusMessage)
	}
}

func TestResumeSlashUnavailableAndBusyStates(t *testing.T) {
	sub := &nopSubmitter{}
	m := newSessionsSlashModelWithOptions(sub, nil, nil)
	m = enterSlashDispatchBehavior(t, m, "/resume sess-missing")
	if sub.calls != 0 {
		t.Fatalf("/resume without adapter reached Submitter %d time(s), want 0", sub.calls)
	}
	if !strings.Contains(m.statusMessage, "resume: session switch unavailable") {
		t.Fatalf("/resume without adapter status = %q, want unavailable", m.statusMessage)
	}

	m = newSessionsSlashModelWithOptions(sub, nil, func(context.Context, string) (SessionResumeResult, error) {
		t.Fatal("resume adapter should not run while a turn is active")
		return SessionResumeResult{}, nil
	})
	m.frame.Phase = kernel.PhaseStreaming
	m = enterSlashDispatchBehavior(t, m, "/resume sess-busy")
	if !strings.Contains(m.statusMessage, "interrupt the current turn before trying to switch sessions") {
		t.Fatalf("/resume busy status = %q, want interrupt guidance", m.statusMessage)
	}
}

func TestSessionsSlashUnavailableAndEmptyStates(t *testing.T) {
	sub := &nopSubmitter{}
	m := newSessionsSlashModel(sub, nil)
	m = enterSlashDispatchBehavior(t, m, "/sessions")
	if sub.calls != 0 {
		t.Fatalf("/sessions without adapter reached Submitter %d time(s), want 0", sub.calls)
	}
	if !strings.Contains(m.statusMessage, "sessions: directory unavailable") {
		t.Fatalf("/sessions without adapter status = %q, want directory unavailable", m.statusMessage)
	}
	if m.transientPage != nil {
		t.Fatalf("/sessions without adapter opened page %+v, want nil", *m.transientPage)
	}

	m = newSessionsSlashModel(sub, func(limit int) ([]SessionDirectoryEntry, error) { return nil, nil })
	m = enterSlashDispatchBehavior(t, m, "/sessions")
	if !strings.Contains(m.statusMessage, "no sessions found") {
		t.Fatalf("/sessions empty status = %q, want no sessions found", m.statusMessage)
	}
	if m.transientPage != nil {
		t.Fatalf("/sessions empty opened page %+v, want nil", *m.transientPage)
	}
}

func TestSessionsSlashCompletionsMarkResumeAvailable(t *testing.T) {
	completions := HermesSlashCommandCompletions("/res")
	for _, completion := range completions {
		if completion.Name != "resume" {
			continue
		}
		if !completion.Available {
			t.Fatalf("completion %+v marked unavailable, want available after native picker port", completion)
		}
		return
	}
	t.Fatalf("HermesSlashCommandCompletions(/res) = %+v, want resume", completions)
}

func assertSessionsPageContains(t *testing.T, m Model, want ...string) {
	t.Helper()
	if m.transientPage == nil {
		t.Fatal("sessions slash did not open transient page")
	}
	if m.transientPage.Title != "Sessions" {
		t.Fatalf("page title = %q, want Sessions", m.transientPage.Title)
	}
	for _, item := range want {
		if !strings.Contains(m.transientPage.Body, item) {
			t.Fatalf("page body missing %q:\n%s", item, m.transientPage.Body)
		}
	}
}

func newSessionsSlashModel(sub *nopSubmitter, directory SessionDirectoryFunc) Model {
	return newSessionsSlashModelWithOptions(sub, directory, nil)
}

func newSessionsSlashModelWithOptions(sub *nopSubmitter, directory SessionDirectoryFunc, resume SessionResumeFunc) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, SessionID: "sess-current"}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true, SessionDirectory: directory, SessionResume: resume})
	m.frame = kernel.RenderFrame{Phase: kernel.PhaseIdle, SessionID: "sess-current"}
	m.width = 96
	m.height = 28
	return m
}
