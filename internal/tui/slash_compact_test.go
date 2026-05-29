package tui

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestCompactSlashTogglesTranscriptRenderingWithoutSubmitting(t *testing.T) {
	sub := &nopSubmitter{}
	m := newCompactSlashModel(sub, false)
	m.width = 96
	m.height = 28
	m.frame.History = compactSlashHistory()

	full := m.View()
	if !strings.Contains(full, "───") {
		t.Fatalf("full transcript view missing turn separator before /compact on:\n%s", full)
	}

	m = enterSlashDispatchBehavior(t, m, "/compact on")

	if sub.calls != 0 {
		t.Fatalf("/compact reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.editor.Value(); got != "" {
		t.Fatalf("editor value after /compact = %q, want cleared", got)
	}
	if !m.compactTranscript {
		t.Fatal("compactTranscript = false after /compact on, want true")
	}
	if !strings.Contains(m.statusMessage, "compact on") {
		t.Fatalf("status after /compact on = %q, want compact on", m.statusMessage)
	}
	compact := m.View()
	if strings.Contains(compact, "───") {
		t.Fatalf("compact transcript still rendered turn separator:\n%s", compact)
	}
	if strings.Contains(m.statusMessage, "recognized but unavailable") {
		t.Fatalf("/compact fell through to unavailable fallback: %q", m.statusMessage)
	}

	m = enterSlashDispatchBehavior(t, m, "/compact off")
	if m.compactTranscript {
		t.Fatal("compactTranscript = true after /compact off, want false")
	}
	if !strings.Contains(m.statusMessage, "compact off") {
		t.Fatalf("status after /compact off = %q, want compact off", m.statusMessage)
	}
	if restored := m.View(); !strings.Contains(restored, "───") {
		t.Fatalf("full transcript view missing turn separator after /compact off:\n%s", restored)
	}
}

func TestCompactSlashToggleAndUsage(t *testing.T) {
	tests := []struct {
		name        string
		initial     bool
		input       string
		wantCompact bool
		wantStatus  string
	}{
		{name: "bare toggles on", input: "/compact", wantCompact: true, wantStatus: "compact on"},
		{name: "toggle flips off", initial: true, input: "/compact toggle", wantCompact: false, wantStatus: "compact off"},
		{name: "invalid usage", input: "/compact maybe", wantCompact: false, wantStatus: "usage: /compact [on|off|toggle]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := &nopSubmitter{}
			m := enterSlashDispatchBehavior(t, newCompactSlashModel(sub, tt.initial), tt.input)
			if sub.calls != 0 {
				t.Fatalf("%s reached Submitter %d time(s), want 0", tt.input, sub.calls)
			}
			if m.compactTranscript != tt.wantCompact {
				t.Fatalf("compactTranscript after %s = %v, want %v", tt.input, m.compactTranscript, tt.wantCompact)
			}
			if !strings.Contains(m.statusMessage, tt.wantStatus) {
				t.Fatalf("status after %s = %q, want %q", tt.input, m.statusMessage, tt.wantStatus)
			}
		})
	}
}

func TestCompactSlashCompletionsAndBusyAvailability(t *testing.T) {
	completions := HermesSlashCommandCompletions("/com")
	for _, completion := range completions {
		if completion.Name == "compact" {
			if !completion.Available {
				t.Fatalf("completion %+v marked unavailable, want available", completion)
			}
			goto foundCompletion
		}
	}
	t.Fatalf("HermesSlashCommandCompletions(/com) = %+v, want compact", completions)

foundCompletion:
	busy := NewDefaultSlashRegistry().BusyAvailableSlashes()
	for _, name := range busy {
		if name == "compact" {
			return
		}
	}
	t.Fatalf("BusyAvailableSlashes() = %v, want compact", busy)
}

func TestCompactSlashKeepsTinyTerminalAutoCompact(t *testing.T) {
	m := newCompactSlashModel(&nopSubmitter{}, false)
	m.width = 5
	m.height = 12
	m.frame.History = compactSlashHistory()

	got := m.View()
	if strings.Contains(got, "───") {
		t.Fatalf("tiny terminal should remain auto-compact even when /compact is off:\n%s", got)
	}
}

func newCompactSlashModel(sub *nopSubmitter, compact bool) Model {
	if sub == nil {
		sub = &nopSubmitter{}
	}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Seq: 1}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true, CompactTranscript: compact})
	m.frame.Phase = kernel.PhaseIdle
	m.width = 96
	m.height = 28
	return m
}

func compactSlashHistory() []llm.Message {
	return []llm.Message{
		{Role: "user", Content: "first question"},
		{Role: "assistant", Content: "first answer"},
		{Role: "user", Content: "second question with enough words to make compact mode visibly collapse the transcript row into one line"},
		{Role: "assistant", Content: "second answer"},
	}
}
