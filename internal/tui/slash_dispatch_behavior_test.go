package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestHermesSlashDispatchBehavior_LocalHandlersStillRun(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		withHistory bool
		wantStatus  string
	}{
		{name: "save", input: "/save", wantStatus: "save: no conversation"},
		{name: "branch", input: "/branch branch-name", wantStatus: "branch: no conversation"},
		{name: "browser", input: "/browser status", wantStatus: "browser:"},
		{name: "mouse", input: "/mouse on", wantStatus: "mouse tracking on"},
		{name: "scroll", input: "/scroll off", wantStatus: "mouse tracking off"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := &nopSubmitter{}
			m := newSlashDispatchBehaviorModel(sub)
			if tt.withHistory {
				m.frame.History = []hermes.Message{{Role: "user", Content: "hello"}}
				m.frame.SessionID = "sess-parent"
			}

			m = enterSlashDispatchBehavior(t, m, tt.input)

			if sub.calls != 0 {
				t.Fatalf("%s reached Submitter %d time(s), want 0", tt.input, sub.calls)
			}
			if got := m.editor.Value(); got != "" {
				t.Fatalf("editor value after %s = %q, want cleared", tt.input, got)
			}
			if !strings.Contains(m.statusMessage, tt.wantStatus) {
				t.Fatalf("status after %s = %q, want it to contain %q", tt.input, m.statusMessage, tt.wantStatus)
			}
		})
	}
}

func TestHermesSlashDispatchBehavior_QuitExitsLocally(t *testing.T) {
	for _, input := range []string{"/quit", "/exit"} {
		t.Run(input, func(t *testing.T) {
			sub := &nopSubmitter{}
			m := newSlashDispatchBehaviorModel(sub)
			m.editor.SetValue(input)

			next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			updated, ok := next.(Model)
			if !ok {
				t.Fatalf("Update returned %T, want tui.Model", next)
			}
			if sub.calls != 0 {
				t.Fatalf("%s reached Submitter %d time(s), want 0", input, sub.calls)
			}
			if got := updated.editor.Value(); got != "" {
				t.Fatalf("editor value after %s = %q, want cleared", input, got)
			}
			if !cmdEmitsQuit(cmd) {
				t.Fatalf("%s did not emit tea.Quit", input)
			}
		})
	}
}

func TestHermesSlashDispatchBehavior_KnownUnhandledCommandsNeverSubmit(t *testing.T) {
	for _, input := range []string{
		"/provider openrouter",
		"/skills",
		"/details tools",
		"/tools list",
		"/history",
		"/status",
		"/title new name",
		"/rollback",
		"/queue later",
	} {
		t.Run(input, func(t *testing.T) {
			sub := &nopSubmitter{}
			m := enterSlashDispatchBehavior(t, newSlashDispatchBehaviorModel(sub), input)

			if sub.calls != 0 {
				t.Fatalf("%s reached Submitter %d time(s), want 0", input, sub.calls)
			}
			if got := m.editor.Value(); got != "" {
				t.Fatalf("editor value after %s = %q, want cleared", input, got)
			}
			status := strings.ToLower(m.statusMessage)
			if !strings.Contains(status, "recognized") {
				t.Fatalf("status after %s = %q, want recognized-command evidence", input, m.statusMessage)
			}
			if !strings.Contains(status, "native tui") && !strings.Contains(status, "gateway") {
				t.Fatalf("status after %s = %q, want native TUI or gateway degraded-mode evidence", input, m.statusMessage)
			}
		})
	}
}

func TestHermesSlashDispatchBehavior_UnknownAndAmbiguousSlashGuidance(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantStatus string
	}{
		{name: "unknown", input: "/no-such-command-xyzzy", wantStatus: "unknown command"},
		{name: "ambiguous", input: "/s", wantStatus: "ambiguous command"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := &nopSubmitter{}
			m := enterSlashDispatchBehavior(t, newSlashDispatchBehaviorModel(sub), tt.input)

			if sub.calls != 0 {
				t.Fatalf("%s reached Submitter %d time(s), want 0", tt.input, sub.calls)
			}
			if got := m.editor.Value(); got != "" {
				t.Fatalf("editor value after %s = %q, want cleared", tt.input, got)
			}
			if !strings.Contains(strings.ToLower(m.statusMessage), tt.wantStatus) {
				t.Fatalf("status after %s = %q, want %q guidance", tt.input, m.statusMessage, tt.wantStatus)
			}
		})
	}
}

func TestHermesSlashDispatchBehavior_MutatingCommandsDoNotFallback(t *testing.T) {
	mutating := []string{
		"/background run later",
		"/branch branch-name",
		"/browser status",
		"/busy queue",
		"/fast",
		"/model gpt-5.2",
		"/new",
		"/queue later",
		"/reasoning high",
		"/rollback",
		"/stop",
		"/title new title",
		"/tools disable terminal",
		"/undo",
		"/verbose",
		"/voice",
		"/yolo",
	}
	for _, input := range mutating {
		t.Run(input, func(t *testing.T) {
			sub := &nopSubmitter{}
			m := enterSlashDispatchBehavior(t, newSlashDispatchBehaviorModel(sub), input)

			if sub.calls != 0 {
				t.Fatalf("%s reached Submitter %d time(s), want 0", input, sub.calls)
			}
			if got := m.editor.Value(); got != "" {
				t.Fatalf("editor value after %s = %q, want cleared", input, got)
			}
			if strings.TrimSpace(m.statusMessage) == "" {
				t.Fatalf("status after %s is empty, want visible routing/degraded evidence", input)
			}
		})
	}
}

func newSlashDispatchBehaviorModel(sub *nopSubmitter) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true})
	m.frame.Phase = kernel.PhaseIdle
	return m
}

func enterSlashDispatchBehavior(t *testing.T, m Model, input string) Model {
	t.Helper()
	m.editor.SetValue(input)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", next)
	}
	runTestCmd(t, cmd)
	return updated
}

func cmdEmitsQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	switch msg := cmd().(type) {
	case tea.QuitMsg:
		return true
	case tea.BatchMsg:
		for _, nested := range msg {
			if cmdEmitsQuit(nested) {
				return true
			}
		}
	}
	return false
}
