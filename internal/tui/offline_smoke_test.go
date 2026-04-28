package tui

import (
	"strings"
	"sync/atomic"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestOfflineSmokeEnterDoesNotSubmitAndRendersLocalTranscript(t *testing.T) {
	var submitCount atomic.Int32
	frames := make(chan kernel.RenderFrame, 4)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle, Model: "hermes-agent", Seq: 1}

	m := NewModelWithOptions(
		frames,
		func(string) { submitCount.Add(1) },
		func() {},
		Options{MouseTracking: true, OfflineSmoke: true},
	)

	m.editor.SetValue("hello offline")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		_ = cmd()
	}
	got, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", updated)
	}

	if n := submitCount.Load(); n != 0 {
		t.Fatalf("Submitter called %d times in offline smoke mode, want 0", n)
	}
	if got.inFlight {
		t.Fatal("inFlight = true after offline smoke submit, want false")
	}
	if got.editor.Value() != "" {
		t.Fatalf("editor value = %q, want reset", got.editor.Value())
	}
	if len(got.frame.History) != 2 {
		t.Fatalf("history len = %d, want local user+assistant transcript: %+v", len(got.frame.History), got.frame.History)
	}
	if got.frame.History[0].Role != "user" || got.frame.History[0].Content != "hello offline" {
		t.Fatalf("user history = %+v, want submitted text", got.frame.History[0])
	}
	if got.frame.History[1].Role != "assistant" || !strings.Contains(got.frame.History[1].Content, "No provider call") {
		t.Fatalf("assistant history = %+v, want offline proof", got.frame.History[1])
	}
	if !strings.Contains(got.statusMessage, "offline") {
		t.Fatalf("statusMessage = %q, want offline evidence", got.statusMessage)
	}
}

func TestFailedFrameResetsInFlightForExit(t *testing.T) {
	frames := make(chan kernel.RenderFrame, 4)
	m := NewModel(frames, func(string) {}, func() {})
	m.inFlight = true

	updated, _ := m.Update(frameMsg(kernel.RenderFrame{
		Phase:     kernel.PhaseFailed,
		Model:     "hermes-agent",
		Seq:       2,
		LastError: "dial tcp 127.0.0.1:8642: connect: connection refused",
	}))
	got, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want tui.Model", updated)
	}
	if got.inFlight {
		t.Fatal("inFlight = true after PhaseFailed frame, want false so Ctrl+C can quit")
	}
}
