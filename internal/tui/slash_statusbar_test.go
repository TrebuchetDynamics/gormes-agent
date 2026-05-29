package tui

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestStatusBarSlashTogglesChromePlacementWithoutSubmitting(t *testing.T) {
	sub := &nopSubmitter{}
	m := newStatusBarSlashModel(sub)

	initial := m.View()
	assertStatusBarBeforePrompt(t, initial)

	m = enterSlashDispatchBehavior(t, m, "/statusbar off")
	if sub.calls != 0 {
		t.Fatalf("/statusbar reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /statusbar = %q, want cleared", got)
	}
	if m.statusBarMode != StatusBarModeOff {
		t.Fatalf("statusBarMode after /statusbar off = %q, want %q", m.statusBarMode, StatusBarModeOff)
	}
	if !strings.Contains(m.statusMessage, "status bar off") {
		t.Fatalf("status after /statusbar off = %q, want off evidence", m.statusMessage)
	}
	if got := m.View(); strings.Contains(got, "─ ready │") {
		t.Fatalf("/statusbar off still rendered the status rule:\n%s", got)
	}
	if strings.Contains(strings.ToLower(m.statusMessage), "recognized") {
		t.Fatalf("/statusbar fell through to fallback: %q", m.statusMessage)
	}

	m = enterSlashDispatchBehavior(t, m, "/statusbar on")
	if m.statusBarMode != StatusBarModeTop {
		t.Fatalf("statusBarMode after /statusbar on = %q, want %q", m.statusBarMode, StatusBarModeTop)
	}
	assertStatusBarBeforePrompt(t, m.View())

	m = enterSlashDispatchBehavior(t, m, "/statusbar bottom")
	if m.statusBarMode != StatusBarModeBottom {
		t.Fatalf("statusBarMode after /statusbar bottom = %q, want %q", m.statusBarMode, StatusBarModeBottom)
	}
	assertStatusBarAfterPrompt(t, m.View())

	m = enterSlashDispatchBehavior(t, m, "/sb top")
	if m.statusBarMode != StatusBarModeTop {
		t.Fatalf("statusBarMode after /sb top = %q, want %q", m.statusBarMode, StatusBarModeTop)
	}
	assertStatusBarBeforePrompt(t, m.View())
}

func TestStatusBarSlashToggleAndUsage(t *testing.T) {
	tests := []struct {
		name     string
		initial  StatusBarMode
		input    string
		wantMode StatusBarMode
		want     string
	}{
		{name: "bare toggles off from top", initial: StatusBarModeTop, input: "/statusbar", wantMode: StatusBarModeOff, want: "status bar off"},
		{name: "toggle restores top from off", initial: StatusBarModeOff, input: "/statusbar toggle", wantMode: StatusBarModeTop, want: "status bar top"},
		{name: "bottom accepted", initial: StatusBarModeTop, input: "/statusbar bottom", wantMode: StatusBarModeBottom, want: "status bar bottom"},
		{name: "invalid usage", initial: StatusBarModeBottom, input: "/statusbar sideways", wantMode: StatusBarModeBottom, want: "usage: /statusbar [on|off|top|bottom|toggle]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newStatusBarSlashModel(&nopSubmitter{})
			m.statusBarMode = tt.initial
			m = enterSlashDispatchBehavior(t, m, tt.input)
			if m.statusBarMode != tt.wantMode {
				t.Fatalf("statusBarMode after %s = %q, want %q", tt.input, m.statusBarMode, tt.wantMode)
			}
			if !strings.Contains(m.statusMessage, tt.want) {
				t.Fatalf("status after %s = %q, want %q", tt.input, m.statusMessage, tt.want)
			}
		})
	}
}

func TestStatusBarSlashCompletionsAndBusyAvailability(t *testing.T) {
	completions := HermesSlashCommandCompletions("/statusb")
	for _, completion := range completions {
		if completion.Name != "statusbar" {
			continue
		}
		if !completion.Available {
			t.Fatalf("completion %+v marked unavailable, want available", completion)
		}
		goto foundCompletion
	}
	t.Fatalf("HermesSlashCommandCompletions(/statusb) = %+v, want statusbar", completions)

foundCompletion:
	got := completionNames(HermesSlashSubcommandCompletions("/statusbar "))
	want := []string{"on", "off", "top", "bottom", "toggle"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("HermesSlashSubcommandCompletions(/statusbar ) = %v, want %v", got, want)
	}

	busy := NewDefaultSlashRegistry().BusyAvailableSlashes()
	for _, name := range busy {
		if name == "statusbar" {
			return
		}
	}
	t.Fatalf("BusyAvailableSlashes() = %v, want statusbar", busy)
}

func newStatusBarSlashModel(sub *nopSubmitter) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frame := kernel.RenderFrame{
		Phase:     kernel.PhaseIdle,
		Model:     "anthropic/claude-sonnet-4-20250514",
		SessionID: "sess-statusbar",
		History: []llm.Message{
			{Role: "user", Content: "show chrome"},
			{Role: "assistant", Content: "chrome visible"},
		},
	}
	frames <- frame
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true})
	m.frame = frame
	m.width = 96
	m.height = 28
	return m
}

func assertStatusBarBeforePrompt(t *testing.T, view string) {
	t.Helper()
	statusIdx := strings.Index(view, "─ ready │")
	promptIdx := strings.LastIndex(view, "❯")
	if statusIdx < 0 || promptIdx < 0 || statusIdx >= promptIdx {
		t.Fatalf("want status bar before prompt, statusIdx=%d promptIdx=%d:\n%s", statusIdx, promptIdx, view)
	}
}

func assertStatusBarAfterPrompt(t *testing.T, view string) {
	t.Helper()
	statusIdx := strings.LastIndex(view, "─ ready │")
	promptIdx := strings.LastIndex(view, "❯")
	if statusIdx < 0 || promptIdx < 0 || promptIdx >= statusIdx {
		t.Fatalf("want status bar after prompt, statusIdx=%d promptIdx=%d:\n%s", statusIdx, promptIdx, view)
	}
}
