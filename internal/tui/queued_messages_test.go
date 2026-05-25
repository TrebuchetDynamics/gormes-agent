package tui

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

// TestQueuedMessages_RemoveAtBounds proves removing a valid index mutates
// the queue and out-of-range indexes are no-ops. Mirrors Hermes useQueue's
// removeAt invariant from ea1012f5.
func TestQueuedMessages_RemoveAtBounds(t *testing.T) {
	t.Run("valid index removes that item only", func(t *testing.T) {
		q := &QueuedMessages{}
		q.Enqueue("a")
		q.Enqueue("b")
		q.Enqueue("c")

		removed := q.RemoveAt(1)
		if !removed {
			t.Fatalf("RemoveAt(1) = false, want true")
		}
		got := q.Items()
		want := []string{"a", "c"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Items() = %v, want %v", got, want)
		}
	})

	t.Run("negative index is a no-op", func(t *testing.T) {
		q := &QueuedMessages{}
		q.Enqueue("a")
		q.Enqueue("b")

		removed := q.RemoveAt(-1)
		if removed {
			t.Fatalf("RemoveAt(-1) = true, want false")
		}
		got := q.Items()
		want := []string{"a", "b"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Items() = %v, want %v", got, want)
		}
	})

	t.Run("index past end is a no-op", func(t *testing.T) {
		q := &QueuedMessages{}
		q.Enqueue("a")
		q.Enqueue("b")

		removed := q.RemoveAt(2)
		if removed {
			t.Fatalf("RemoveAt(2) = true, want false")
		}
		got := q.Items()
		want := []string{"a", "b"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Items() = %v, want %v", got, want)
		}
	})

	t.Run("empty queue refuses any index", func(t *testing.T) {
		q := &QueuedMessages{}
		if q.RemoveAt(0) {
			t.Fatalf("RemoveAt(0) on empty queue = true, want false")
		}
		if got := q.Items(); len(got) != 0 {
			t.Fatalf("Items() = %v, want empty", got)
		}
	})
}

// TestQueuedMessages_CancelEditPreservesQueue proves cancelling an edit
// clears edit state while keeping the original queued text intact. The
// not_ready_when contract for the slice forbids cancel-as-delete.
func TestQueuedMessages_CancelEditPreservesQueue(t *testing.T) {
	q := &QueuedMessages{}
	q.Enqueue("first")
	q.Enqueue("second")
	q.Enqueue("third")

	if !q.SelectEdit(1) {
		t.Fatalf("SelectEdit(1) = false, want true")
	}
	if got, ok := q.EditIndex(); !ok || got != 1 {
		t.Fatalf("EditIndex() = (%d, %v), want (1, true)", got, ok)
	}

	q.CancelEdit()

	if _, ok := q.EditIndex(); ok {
		t.Fatalf("EditIndex() still set after CancelEdit")
	}
	got := q.Items()
	want := []string{"first", "second", "third"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Items() = %v, want %v", got, want)
	}
}

// TestQueuedMessages_DeleteEditingItemRemovesOnlySelected proves deleting
// edit index 1 from [a,b,c] yields [a,c], clears edit state, and does not
// dequeue the head. Tracks Hermes Ctrl+X delete-while-editing semantics.
func TestQueuedMessages_DeleteEditingItemRemovesOnlySelected(t *testing.T) {
	q := &QueuedMessages{}
	q.Enqueue("a")
	q.Enqueue("b")
	q.Enqueue("c")

	if !q.SelectEdit(1) {
		t.Fatalf("SelectEdit(1) = false, want true")
	}

	deleted, ok := q.DeleteEditing()
	if !ok {
		t.Fatalf("DeleteEditing() ok = false, want true")
	}
	if deleted != "b" {
		t.Fatalf("DeleteEditing() deleted = %q, want %q", deleted, "b")
	}
	got := q.Items()
	want := []string{"a", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Items() = %v, want %v", got, want)
	}
	if _, set := q.EditIndex(); set {
		t.Fatalf("EditIndex() still set after DeleteEditing")
	}
}

// TestQueuedMessages_ReplaceEditingItem proves replacing the selected item
// updates only that item and keeps queue order. Mirrors Hermes useQueue
// replaceQ for the editing index path.
func TestQueuedMessages_ReplaceEditingItem(t *testing.T) {
	q := &QueuedMessages{}
	q.Enqueue("alpha")
	q.Enqueue("beta")
	q.Enqueue("gamma")

	if !q.SelectEdit(1) {
		t.Fatalf("SelectEdit(1) = false, want true")
	}

	if !q.ReplaceEditing("BETA-edited") {
		t.Fatalf("ReplaceEditing returned false, want true")
	}

	got := q.Items()
	want := []string{"alpha", "BETA-edited", "gamma"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Items() = %v, want %v", got, want)
	}
	if idx, ok := q.EditIndex(); !ok || idx != 1 {
		t.Fatalf("EditIndex() = (%d, %v), want (1, true)", idx, ok)
	}

	t.Run("replace without selection is a no-op", func(t *testing.T) {
		q := &QueuedMessages{}
		q.Enqueue("alpha")
		q.Enqueue("beta")

		if q.ReplaceEditing("nope") {
			t.Fatalf("ReplaceEditing without selection = true, want false")
		}
		got := q.Items()
		want := []string{"alpha", "beta"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Items() = %v, want %v", got, want)
		}
	})
}

// TestQueuedMessages_WindowCentersEditedItem proves the visible three-row
// window matches Hermes' start/end/lead/tail behaviour for head, middle,
// tail, and empty queues. Tracks
// hermes-agent/ui-tui/src/components/queuedMessages.tsx@ea1012f5:getQueueWindow.
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

func TestQueuedMessages_WindowCentersEditedItem(t *testing.T) {
	cases := []struct {
		name    string
		length  int
		editIdx int
		editing bool
		wantWin QueueWindow
	}{
		{
			name:    "empty queue with no edit collapses to zero window",
			length:  0,
			editing: false,
			wantWin: QueueWindow{Start: 0, End: 0, ShowLead: false, ShowTail: false},
		},
		{
			name:    "no edit selection anchors window at head",
			length:  5,
			editing: false,
			wantWin: QueueWindow{Start: 0, End: 3, ShowLead: false, ShowTail: true},
		},
		{
			name:    "edit at head keeps window at start",
			length:  5,
			editIdx: 0,
			editing: true,
			wantWin: QueueWindow{Start: 0, End: 3, ShowLead: false, ShowTail: true},
		},
		{
			name:    "edit in middle centres window with lead and tail",
			length:  5,
			editIdx: 2,
			editing: true,
			wantWin: QueueWindow{Start: 1, End: 4, ShowLead: true, ShowTail: true},
		},
		{
			name:    "edit at tail clamps window to end",
			length:  5,
			editIdx: 4,
			editing: true,
			wantWin: QueueWindow{Start: 2, End: 5, ShowLead: true, ShowTail: false},
		},
		{
			name:    "queue shorter than window with edit at head shows everything",
			length:  2,
			editIdx: 0,
			editing: true,
			wantWin: QueueWindow{Start: 0, End: 2, ShowLead: false, ShowTail: false},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got QueueWindow
			if tc.editing {
				got = ComputeQueueWindow(tc.length, &tc.editIdx)
			} else {
				got = ComputeQueueWindow(tc.length, nil)
			}
			if got != tc.wantWin {
				t.Fatalf("ComputeQueueWindow(%d, editing=%v idx=%d) = %+v, want %+v",
					tc.length, tc.editing, tc.editIdx, got, tc.wantWin)
			}
		})
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
