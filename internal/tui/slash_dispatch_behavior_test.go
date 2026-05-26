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

func TestHermesSlashDispatchBehavior_RedrawClearsVisibleFrameLocally(t *testing.T) {
	sub := &nopSubmitter{}
	m := newSlashDispatchBehaviorModel(sub)
	m.frame.History = []hermes.Message{{Role: "user", Content: "keep in kernel, clear from view"}}
	m.frame.DraftText = "streaming draft"
	m.frame.LastError = "stale terminal error"
	m.frame.SessionID = "sess-redraw"

	m = enterSlashDispatchBehavior(t, m, "/redraw")

	if sub.calls != 0 {
		t.Fatalf("/redraw reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /redraw = %q, want cleared", got)
	}
	if len(m.frame.History) != 0 || m.frame.DraftText != "" || m.frame.LastError != "" {
		t.Fatalf("/redraw did not clear visible frame: history=%d draft=%q err=%q", len(m.frame.History), m.frame.DraftText, m.frame.LastError)
	}
	if m.frame.SessionID != "sess-redraw" {
		t.Fatalf("/redraw SessionID = %q, want preserved", m.frame.SessionID)
	}
	if !strings.Contains(strings.ToLower(m.statusMessage), "ui redrawn") {
		t.Fatalf("status after /redraw = %q, want ui redrawn", m.statusMessage)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/redraw fell through to unavailable fallback: %q", m.statusMessage)
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

func TestHermesSlashDispatchBehavior_SkillsSlashRunsLocallyWhileBusy(t *testing.T) {
	sub := &nopSubmitter{}
	m := newSlashDispatchBehaviorModel(sub)
	m.inFlight = true
	m.frame.Phase = kernel.PhaseStreaming

	m = enterSlashDispatchBehavior(t, m, "/skills search planner")

	if sub.calls != 0 {
		t.Fatalf("/skills search reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /skills search = %q, want cleared", got)
	}
	status := strings.ToLower(m.statusMessage)
	if !strings.Contains(status, "skill hub search") || strings.Contains(status, "recognized but unavailable") {
		t.Fatalf("status after /skills search = %q, want local skills command output", m.statusMessage)
	}
}

func TestHermesSlashDispatchBehavior_SkillsInstallRunsLocally(t *testing.T) {
	sub := &nopSubmitter{}
	calls := 0
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{
		MouseTracking: true,
		SkillsCommand: func(input string) string {
			calls++
			if input != "/skills install https://example.com/SKILL.md --name tui-skill" {
				t.Fatalf("SkillsCommand input = %q", input)
			}
			return "url_skill_installed: installed tui-skill"
		},
	})
	m.frame.Phase = kernel.PhaseIdle

	m = enterSlashDispatchBehavior(t, m, "/skills install https://example.com/SKILL.md --name tui-skill")

	if calls != 1 {
		t.Fatalf("SkillsCommand calls = %d, want 1", calls)
	}
	if sub.calls != 0 {
		t.Fatalf("/skills install reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /skills install = %q, want cleared", got)
	}
	status := strings.ToLower(m.statusMessage)
	if !strings.Contains(status, "url_skill_installed") || !strings.Contains(status, "tui-skill") {
		t.Fatalf("status after /skills install = %q, want install evidence", m.statusMessage)
	}
}

func TestHermesSlashDispatchBehavior_KnownUnhandledCommandsNeverSubmit(t *testing.T) {
	for _, input := range []string{
		"/provider openrouter",
		"/image ./diagram.png",
		"/tools list",
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
