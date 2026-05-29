package tui

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestQueuedMessages_ActiveTurnQueueModeRendersWithoutSubmitting(t *testing.T) {
	sub := &recordingQueuedSubmitter{}
	m := newQueuedMessageTestModel(sub.submit, nil, Options{BusyInputMode: HermesBusyQueue})
	m.editor.SetValue("follow up after this turn")

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := next.(Model)
	runTestCmd(t, cmd)

	if len(sub.texts) != 0 {
		t.Fatalf("queue mode reached Submitter with %v, want no immediate submit", sub.texts)
	}
	if got := updated.queuedMessages.Items(); !reflect.DeepEqual(got, []string{"follow up after this turn"}) {
		t.Fatalf("queued items = %v, want follow-up draft", got)
	}
	if got := updated.editor.Value(); got != "" {
		t.Fatalf("editor value after queue = %q, want empty", got)
	}

	view := updated.View()
	assertContainsInOrder(t, view, "queued (1)", "follow up after this turn", "─ running", "❯")
	assertRenderedWidthAtMost(t, view, updated.width)
}

func TestQueuedMessages_IdleFrameDrainsFIFOAndClearsWidget(t *testing.T) {
	sub := &recordingQueuedSubmitter{}
	frames := make(chan kernel.RenderFrame, 2)
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{BusyInputMode: HermesBusyQueue})
	m.width = 90
	m.height = 28
	m.inFlight = true
	m.frame = kernel.RenderFrame{Phase: kernel.PhaseStreaming, Model: "anthropic/claude-sonnet-4-20250514", SessionID: "sess-queue-drain"}
	m.queuedMessages.Enqueue("first follow-up")
	m.queuedMessages.Enqueue("second follow-up")

	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle}
	next, cmd := m.Update(frameMsg(kernel.RenderFrame{Phase: kernel.PhaseIdle, Model: m.frame.Model, SessionID: m.frame.SessionID}))
	updated := next.(Model)
	runTestCmd(t, cmd)

	if !reflect.DeepEqual(sub.texts, []string{"first follow-up"}) {
		t.Fatalf("submitted texts after first idle = %v, want first follow-up", sub.texts)
	}
	if got := updated.queuedMessages.Items(); !reflect.DeepEqual(got, []string{"second follow-up"}) {
		t.Fatalf("remaining queue = %v, want second follow-up", got)
	}
	if !updated.inFlight {
		t.Fatalf("inFlight = false after draining first queued item, want true")
	}
	view := updated.View()
	if strings.Contains(view, "first follow-up") {
		t.Fatalf("drained item remained visible:\n%s", view)
	}
	if !strings.Contains(view, "second follow-up") {
		t.Fatalf("remaining queued item not visible:\n%s", view)
	}

	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle}
	next, cmd = updated.Update(frameMsg(kernel.RenderFrame{Phase: kernel.PhaseFailed, Model: m.frame.Model, SessionID: m.frame.SessionID}))
	updated = next.(Model)
	runTestCmd(t, cmd)

	if !reflect.DeepEqual(sub.texts, []string{"first follow-up", "second follow-up"}) {
		t.Fatalf("submitted texts after second terminal frame = %v, want FIFO drain", sub.texts)
	}
	if got := updated.queuedMessages.Items(); len(got) != 0 {
		t.Fatalf("queue after full drain = %v, want empty", got)
	}
	if strings.Contains(updated.View(), "queued (") {
		t.Fatalf("queued widget still visible after full drain:\n%s", updated.View())
	}
}

func TestQueuedMessages_SteerModeUsesSteerSeamAndShowsEvidence(t *testing.T) {
	sub := &recordingQueuedSubmitter{}
	steer := &recordingQueuedSteerer{}
	m := newQueuedMessageTestModel(sub.submit, steer.steer, Options{BusyInputMode: HermesBusySteer})
	m.editor.SetValue("inspect the failing test before editing")

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := next.(Model)
	runTestCmd(t, cmd)

	if len(sub.texts) != 0 {
		t.Fatalf("steer mode reached Submitter with %v, want no follow-up submit", sub.texts)
	}
	if !reflect.DeepEqual(steer.texts, []string{"inspect the failing test before editing"}) {
		t.Fatalf("steer texts = %v, want operator guidance", steer.texts)
	}
	if got := updated.steeringMessages.Items(); !reflect.DeepEqual(got, []string{"inspect the failing test before editing"}) {
		t.Fatalf("visible steering messages = %v, want guidance", got)
	}
	if !strings.Contains(strings.ToLower(updated.statusMessage), "steer") {
		t.Fatalf("statusMessage = %q, want steer evidence", updated.statusMessage)
	}
	view := updated.View()
	assertContainsInOrder(t, view, "steering (1)", "inspect the failing test", "─ running", "❯")

	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseIdle}
	updated.frames = frames
	next, cmd = updated.Update(frameMsg(kernel.RenderFrame{Phase: kernel.PhaseIdle, Model: updated.frame.Model, SessionID: updated.frame.SessionID}))
	updated = next.(Model)
	runTestCmd(t, cmd)
	if got := updated.steeringMessages.Items(); len(got) != 0 {
		t.Fatalf("steering messages after idle = %v, want cleared", got)
	}
	if strings.Contains(updated.View(), "steering (") {
		t.Fatalf("steering widget still visible after idle:\n%s", updated.View())
	}
}

func TestQueuedMessages_AltEnterAndShiftEnterNeverQueueOrSubmit(t *testing.T) {
	sub := &recordingQueuedSubmitter{}
	steer := &recordingQueuedSteerer{}
	m := newQueuedMessageTestModel(sub.submit, steer.steer, Options{BusyInputMode: HermesBusyQueue})
	m.editor.SetValue("multi-line draft")

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter, Alt: true})
	updated := next.(Model)
	runTestCmd(t, cmd)
	if len(sub.texts) != 0 || len(steer.texts) != 0 || updated.queuedMessages.Len() != 0 {
		t.Fatalf("Alt+Enter submitted=%v steered=%v queued=%v; want no delivery", sub.texts, steer.texts, updated.queuedMessages.Items())
	}

	next, cmd = updated.Update([]byte("\x1b[13;2u"))
	updated = next.(Model)
	runTestCmd(t, cmd)
	if len(sub.texts) != 0 || len(steer.texts) != 0 || updated.queuedMessages.Len() != 0 {
		t.Fatalf("Shift+Enter submitted=%v steered=%v queued=%v; want no delivery", sub.texts, steer.texts, updated.queuedMessages.Items())
	}
	if !strings.Contains(updated.editor.Value(), "\n") {
		t.Fatalf("editor value after multiline keys = %q, want newline retained", updated.editor.Value())
	}
}

func newQueuedMessageTestModel(submit Submitter, steer Steerer, opts Options) Model {
	frames := make(chan kernel.RenderFrame, 1)
	opts.Steer = steer
	m := NewModelWithOptions(frames, submit, func() {}, opts)
	m.width = 90
	m.height = 28
	m.inFlight = true
	m.frame = kernel.RenderFrame{
		Phase:     kernel.PhaseStreaming,
		Model:     "anthropic/claude-sonnet-4-20250514",
		SessionID: "sess-queued-widget",
	}
	return m
}

type recordingQueuedSubmitter struct {
	texts []string
}

func (r *recordingQueuedSubmitter) submit(text string) {
	r.texts = append(r.texts, text)
}

type recordingQueuedSteerer struct {
	texts []string
}

func (r *recordingQueuedSteerer) steer(text string) {
	r.texts = append(r.texts, text)
}
