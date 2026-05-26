package tui

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestQueueSlashAppendsVisibleQueueWithoutModelLeak(t *testing.T) {
	sub := &recordingSubmitter{}
	m := newSlashDispatchBehaviorModel((*nopSubmitter)(nil))
	m.submit = sub.submit
	m.inFlight = true
	m.frame.Phase = kernel.PhaseStreaming

	m = enterSlashDispatchBehavior(t, m, "/queue follow up after tools")
	if sub.calls != 0 {
		t.Fatalf("/queue reached Submitter %d time(s), want 0", sub.calls)
	}
	if got := m.queuedMessages.Len(); got != 1 {
		t.Fatalf("queued messages = %d, want 1", got)
	}
	if got := m.queuedMessages.Items()[0]; got != "follow up after tools" {
		t.Fatalf("queued text = %q, want stripped prompt", got)
	}
	if strings.Contains(strings.TrimSpace(m.statusMessage), "recognized") {
		t.Fatalf("/queue fell through to unavailable evidence: %q", m.statusMessage)
	}
	if !strings.Contains(m.statusMessage, "queued") {
		t.Fatalf("status after /queue = %q, want queued evidence", m.statusMessage)
	}

	m = enterSlashDispatchBehavior(t, m, "/queue")
	if !strings.Contains(m.statusMessage, "1 queued message(s)") {
		t.Fatalf("status after /queue status = %q, want queue depth", m.statusMessage)
	}
}

func TestQueueSlashDrainsAfterTurnSettles(t *testing.T) {
	sub := &recordingSubmitter{}
	frames := make(chan kernel.RenderFrame, 1)
	frames <- kernel.RenderFrame{Phase: kernel.PhaseStreaming, Seq: 1}
	m := NewModelWithOptions(frames, sub.submit, func() {}, Options{MouseTracking: true})
	m.frame.Phase = kernel.PhaseStreaming
	m.inFlight = true

	m = enterSlashDispatchBehavior(t, m, "/queue next turn")
	if sub.calls != 0 {
		t.Fatalf("/queue submit calls before drain = %d, want 0", sub.calls)
	}

	next, cmd := m.Update(frameMsg(kernel.RenderFrame{Phase: kernel.PhaseIdle, DraftText: "done"}))
	updated, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", next)
	}
	runTestCmd(t, cmd)
	if sub.calls != 1 || sub.texts[0] != "next turn" {
		t.Fatalf("drained submits = %d %#v, want next turn", sub.calls, sub.texts)
	}
	if got := updated.queuedMessages.Len(); got != 0 {
		t.Fatalf("queued messages after drain = %d, want 0", got)
	}
	if !strings.Contains(updated.statusMessage, "queued follow-up sent") {
		t.Fatalf("status after drain = %q, want sent evidence", updated.statusMessage)
	}
}

func TestQueueSlashEmptyReportsDepthEvenIdle(t *testing.T) {
	sub := &recordingSubmitter{}
	m := newSlashDispatchBehaviorModel((*nopSubmitter)(nil))
	m.submit = sub.submit
	m.queuedMessages.Enqueue("already queued")

	m = enterSlashDispatchBehavior(t, m, "/queue   ")
	if sub.calls != 0 {
		t.Fatalf("/queue status reached Submitter %d time(s), want 0", sub.calls)
	}
	if !strings.Contains(m.statusMessage, "1 queued message(s)") {
		t.Fatalf("status after /queue = %q, want queue depth", m.statusMessage)
	}
}
