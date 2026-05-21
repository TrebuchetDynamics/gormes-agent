package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

type recordingSessionReset struct {
	calls int
	err   error
}

func (r *recordingSessionReset) call() error {
	r.calls++
	return r.err
}

func TestSessionSlashClearAndNewResetWithoutSubmitting(t *testing.T) {
	for _, tt := range []struct {
		input      string
		wantStatus string
	}{
		{input: "/clear", wantStatus: "session cleared"},
		{input: "/new", wantStatus: "new session started"},
	} {
		t.Run(tt.input, func(t *testing.T) {
			reset := &recordingSessionReset{}
			sub := &nopSubmitter{}
			m := newSessionResetModel(sub, reset.call)
			m.frame.History = []hermes.Message{{Role: "user", Content: "hello"}, {Role: "assistant", Content: "hi"}}
			m.frame.DraftText = "draft"
			m.frame.LastError = "boom"
			m.frame.SessionID = "sess-frame"
			m.sessionID = "sess-local"

			m = enterSlashDispatchBehavior(t, m, tt.input)

			if sub.calls != 0 {
				t.Fatalf("%s reached Submitter %d time(s), want 0", tt.input, sub.calls)
			}
			if reset.calls != 1 {
				t.Fatalf("SessionResetFunc calls = %d, want 1", reset.calls)
			}
			if got := m.editor.Value(); got != "" {
				t.Fatalf("editor value after %s = %q, want cleared", tt.input, got)
			}
			if len(m.frame.History) != 0 || m.frame.DraftText != "" || m.frame.LastError != "" || m.frame.SessionID != "" || m.sessionID != "" {
				t.Fatalf("visible session state not cleared after %s: frame=%+v sessionID=%q", tt.input, m.frame, m.sessionID)
			}
			if !strings.Contains(m.statusMessage, tt.wantStatus) {
				t.Fatalf("status after %s = %q, want %q", tt.input, m.statusMessage, tt.wantStatus)
			}
			if strings.Contains(m.statusMessage, "recognized but unavailable") {
				t.Fatalf("%s fell through to unavailable fallback: %q", tt.input, m.statusMessage)
			}
		})
	}
}

func TestSessionSlashResetUnavailableAndErrorsDoNotLeak(t *testing.T) {
	for _, tt := range []struct {
		name       string
		input      string
		reset      *recordingSessionReset
		wantStatus string
		wantCalls  int
	}{
		{name: "missing reset seam", input: "/clear", wantStatus: "clear: reset unavailable", wantCalls: 0},
		{name: "reset error", input: "/new", reset: &recordingSessionReset{err: errors.New("db locked")}, wantStatus: "new: reset failed: db locked", wantCalls: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sub := &nopSubmitter{}
			var resetFn SessionResetFunc
			if tt.reset != nil {
				resetFn = tt.reset.call
			}
			m := newSessionResetModel(sub, resetFn)
			m.frame.History = []hermes.Message{{Role: "user", Content: "keep me"}}
			m.frame.SessionID = "sess-frame"

			m = enterSlashDispatchBehavior(t, m, tt.input)

			if sub.calls != 0 {
				t.Fatalf("%s reached Submitter %d time(s), want 0", tt.input, sub.calls)
			}
			if tt.reset != nil && tt.reset.calls != tt.wantCalls {
				t.Fatalf("SessionResetFunc calls = %d, want %d", tt.reset.calls, tt.wantCalls)
			}
			if !strings.Contains(m.statusMessage, tt.wantStatus) {
				t.Fatalf("status after %s = %q, want %q", tt.input, m.statusMessage, tt.wantStatus)
			}
			if len(m.frame.History) != 1 || m.frame.SessionID != "sess-frame" {
				t.Fatalf("failed reset should preserve visible session, got frame=%+v", m.frame)
			}
		})
	}
}

func TestSessionSlashResetRejectedWhileTurnRunning(t *testing.T) {
	reset := &recordingSessionReset{}
	sub := &nopSubmitter{}
	m := newSessionResetModel(sub, reset.call)
	m.inFlight = true
	m.frame.Phase = kernel.PhaseStreaming
	m.frame.History = []hermes.Message{{Role: "user", Content: "keep me"}}

	m = enterSlashDispatchBehavior(t, m, "/new")

	if sub.calls != 0 {
		t.Fatalf("/new reached Submitter %d time(s), want 0", sub.calls)
	}
	if reset.calls != 0 {
		t.Fatalf("SessionResetFunc calls = %d, want 0 while turn is running", reset.calls)
	}
	if !strings.Contains(m.statusMessage, "interrupt the current turn before trying to switch sessions") {
		t.Fatalf("status = %q, want busy session-switch guidance", m.statusMessage)
	}
	if len(m.frame.History) != 1 {
		t.Fatalf("running reset should preserve history, got %+v", m.frame.History)
	}
}

func TestSessionSlashCompletionsMarkClearAndNewAvailable(t *testing.T) {
	for _, input := range []string{"/cle", "/ne"} {
		t.Run(input, func(t *testing.T) {
			completions := HermesSlashCommandCompletions(input)
			if len(completions) == 0 {
				t.Fatalf("HermesSlashCommandCompletions(%q) returned no completions", input)
			}
			for _, completion := range completions {
				if completion.Name == "clear" || completion.Name == "new" {
					if !completion.Available {
						t.Fatalf("completion %+v marked unavailable, want available", completion)
					}
					return
				}
			}
			t.Fatalf("HermesSlashCommandCompletions(%q) = %+v, want clear/new", input, completions)
		})
	}
}

func newSessionResetModel(sub *nopSubmitter, reset SessionResetFunc) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true, SessionReset: reset})
	m.frame.Phase = kernel.PhaseIdle
	return m
}

func TestSessionSlashRegistryConsumesClearAndNew(t *testing.T) {
	for _, input := range []string{"/clear", "/new"} {
		t.Run(input, func(t *testing.T) {
			res := NewDefaultSlashRegistry().Dispatch(input, &Model{sessionReset: func() error { return nil }})
			if !res.Handled {
				t.Fatalf("Dispatch(%q) Handled = false, want true", input)
			}
			if strings.Contains(res.StatusMessage, "recognized but unavailable") {
				t.Fatalf("Dispatch(%q) fell through to fallback: %q", input, res.StatusMessage)
			}
		})
	}
}

var _ tea.Model = Model{}
