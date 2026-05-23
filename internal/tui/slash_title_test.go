package tui

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestTitleSlashGetsAndSetsSessionTitleWithoutSubmitting(t *testing.T) {
	sub := &nopSubmitter{}
	var calls []titleSlashCall
	m := newTitleSlashModel(sub, func(sessionID, title string) (SessionTitleResult, error) {
		calls = append(calls, titleSlashCall{sessionID: sessionID, title: title})
		if title == "" {
			return SessionTitleResult{Title: "demo title"}, nil
		}
		return SessionTitleResult{Title: title}, nil
	})

	m = enterSlashDispatchBehavior(t, m, "/title")
	if sub.calls != 0 {
		t.Fatalf("/title reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /title = %q, want cleared", got)
	}
	if len(calls) != 1 || calls[0] != (titleSlashCall{sessionID: "sess-title"}) {
		t.Fatalf("/title calls = %+v, want get current title for sess-title", calls)
	}
	if !strings.Contains(m.statusMessage, "title: demo title") {
		t.Fatalf("/title status = %q, want current title", m.statusMessage)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/title fell through to fallback: %q", m.statusMessage)
	}

	m = enterSlashDispatchBehavior(t, m, "/title my   title")
	if sub.calls != 0 {
		t.Fatalf("/title set reached Submitter %d time(s), want 0", sub.calls)
	}
	if len(calls) != 2 || calls[1] != (titleSlashCall{sessionID: "sess-title", title: "my title"}) {
		t.Fatalf("/title set calls = %+v, want collapsed multi-word title", calls)
	}
	if !strings.Contains(m.statusMessage, "session title set: my title") {
		t.Fatalf("/title set status = %q, want set confirmation", m.statusMessage)
	}
}

func TestTitleSlashRequiresActiveSessionAndAdapter(t *testing.T) {
	sub := &nopSubmitter{}
	called := false
	m := newTitleSlashModel(sub, func(sessionID, title string) (SessionTitleResult, error) {
		called = true
		return SessionTitleResult{Title: title}, nil
	})
	m.frame.SessionID = ""

	m = enterSlashDispatchBehavior(t, m, "/title new name")
	if sub.calls != 0 {
		t.Fatalf("/title without session reached Submitter %d time(s), want 0", sub.calls)
	}
	if called {
		t.Fatal("/title without an active session called SessionTitle adapter")
	}
	if !strings.Contains(m.statusMessage, "no active session") {
		t.Fatalf("/title without session status = %q, want no active session", m.statusMessage)
	}

	m = newTitleSlashModel(sub, nil)
	m = enterSlashDispatchBehavior(t, m, "/title new name")
	if !strings.Contains(m.statusMessage, "title: session title unavailable") {
		t.Fatalf("/title without adapter status = %q, want unavailable evidence", m.statusMessage)
	}
}

func TestTitleSlashBusyAvailability(t *testing.T) {
	completions := HermesSlashCommandCompletions("/tit")
	for _, completion := range completions {
		if completion.Name != "title" {
			continue
		}
		if !completion.Available {
			t.Fatalf("completion %+v marked unavailable, want available", completion)
		}
		goto foundCompletion
	}
	t.Fatalf("HermesSlashCommandCompletions(/tit) = %+v, want title", completions)

foundCompletion:
	busy := NewDefaultSlashRegistry().BusyAvailableSlashes()
	for _, name := range busy {
		if name == "title" {
			return
		}
	}
	t.Fatalf("BusyAvailableSlashes() = %v, want title", busy)
}

type titleSlashCall struct {
	sessionID string
	title     string
}

func newTitleSlashModel(sub *nopSubmitter, title SessionTitleFunc) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frame := kernel.RenderFrame{Phase: kernel.PhaseIdle, SessionID: "sess-title"}
	frames <- frame
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true, SessionTitle: title})
	m.frame = frame
	m.width = 96
	m.height = 28
	return m
}
