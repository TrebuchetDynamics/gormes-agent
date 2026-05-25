package tui

import (
	"fmt"
	"strings"
)

// QueueWindowSize is the number of queued messages rendered per frame. Tracks
// hermes-agent/ui-tui/src/components/queuedMessages.tsx@ea1012f5 (QUEUE_WINDOW=3).
const QueueWindowSize = 3

// QueuedMessages is a pure helper that owns the buffer of pending operator
// turns plus the optional edit selection. It is deliberately isolated from
// Bubble Tea wiring: callers manipulate the buffer directly, then read Items
// and ComputeQueueWindow when rendering. Tracks Hermes useQueue from
// ui-tui/src/hooks/useQueue.ts@ea1012f5.
type QueuedMessages struct {
	items   []string
	editIdx int
	editing bool
}

// Enqueue appends text to the tail of the queue.
func (q *QueuedMessages) Enqueue(text string) {
	q.items = append(q.items, text)
}

// Dequeue removes and returns the head of the queue. The second return value
// reports whether the queue had any item to dequeue.
func (q *QueuedMessages) Dequeue() (string, bool) {
	if len(q.items) == 0 {
		return "", false
	}
	head := q.items[0]
	q.items = append(q.items[:0:0], q.items[1:]...)
	if q.editing {
		switch {
		case q.editIdx == 0:
			q.clearEdit()
		case q.editIdx > 0:
			q.editIdx--
		}
	}
	return head, true
}

// Items returns a copy of the queue contents in FIFO order. The slice is
// owned by the caller; mutating it does not affect the queue.
func (q *QueuedMessages) Items() []string {
	if len(q.items) == 0 {
		return nil
	}
	out := make([]string, len(q.items))
	copy(out, q.items)
	return out
}

// Len returns the number of queued items.
func (q *QueuedMessages) Len() int {
	return len(q.items)
}

// RemoveAt removes the item at index i. It returns true when the index was
// in range and the queue mutated; out-of-range indexes are silent no-ops so
// callers can route Ctrl+X bindings through this helper without bounds
// checks. If the removed index matched the edit selection, edit state is
// cleared; if it preceded the selection, the edit index slides left so it
// keeps pointing at the same logical item.
func (q *QueuedMessages) RemoveAt(i int) bool {
	if i < 0 || i >= len(q.items) {
		return false
	}
	q.items = append(q.items[:i], q.items[i+1:]...)
	if q.editing {
		switch {
		case q.editIdx == i:
			q.clearEdit()
		case q.editIdx > i:
			q.editIdx--
		}
	}
	return true
}

// SelectEdit marks index i as the item under edit. Returns true on success;
// out-of-range indexes leave the queue and prior edit state untouched.
func (q *QueuedMessages) SelectEdit(i int) bool {
	if i < 0 || i >= len(q.items) {
		return false
	}
	q.editIdx = i
	q.editing = true
	return true
}

// EditIndex returns the current edit selection. The second return value is
// false when nothing is selected.
func (q *QueuedMessages) EditIndex() (int, bool) {
	if !q.editing {
		return 0, false
	}
	return q.editIdx, true
}

// CancelEdit clears the edit selection without touching queue contents. The
// not_ready_when contract for this slice forbids cancel-as-delete: only
// DeleteEditing or RemoveAt may remove queued text.
func (q *QueuedMessages) CancelEdit() {
	q.clearEdit()
}

// ReplaceEditing replaces the queued text at the edit index with text. The
// edit selection itself is preserved so the operator can keep iterating on
// the same slot. Returns false when no edit is in progress.
func (q *QueuedMessages) ReplaceEditing(text string) bool {
	if !q.editing {
		return false
	}
	if q.editIdx < 0 || q.editIdx >= len(q.items) {
		q.clearEdit()
		return false
	}
	q.items[q.editIdx] = text
	return true
}

// DeleteEditing removes the item currently under edit and clears edit state.
// The deleted text and ok=true are returned on success; when nothing is
// selected the queue is untouched and ok=false.
func (q *QueuedMessages) DeleteEditing() (string, bool) {
	if !q.editing {
		return "", false
	}
	idx := q.editIdx
	if idx < 0 || idx >= len(q.items) {
		q.clearEdit()
		return "", false
	}
	deleted := q.items[idx]
	q.items = append(q.items[:idx], q.items[idx+1:]...)
	q.clearEdit()
	return deleted, true
}

func (q *QueuedMessages) clearEdit() {
	q.editIdx = 0
	q.editing = false
}

// QueueWindow describes the visible slice of the queued buffer for rendering.
// Start is inclusive, End is exclusive. ShowLead/ShowTail signal the operator
// that hidden items exist before/after the window so the caller can render
// the leading "…" and trailing "…and N more" affordances. Tracks Hermes'
// getQueueWindow shape exactly.
type QueueWindow struct {
	Start    int
	End      int
	ShowLead bool
	ShowTail bool
}

// ComputeQueueWindow returns the three-row visible window for a queue of
// length n. When editIdx is nil the window anchors at the head; otherwise
// the window centres around the edited item but never advances past
// max(0, n-QueueWindowSize). Mirrors the start/end/showLead/showTail formula
// from hermes-agent/ui-tui/src/components/queuedMessages.tsx@ea1012f5.
func ComputeQueueWindow(n int, editIdx *int) QueueWindow {
	if n <= 0 {
		return QueueWindow{}
	}

	maxStart := n - QueueWindowSize
	if maxStart < 0 {
		maxStart = 0
	}

	start := 0
	if editIdx != nil {
		candidate := *editIdx - 1
		if candidate < 0 {
			candidate = 0
		}
		if candidate > maxStart {
			candidate = maxStart
		}
		start = candidate
	}

	end := start + QueueWindowSize
	if end > n {
		end = n
	}

	return QueueWindow{
		Start:    start,
		End:      end,
		ShowLead: start > 0,
		ShowTail: end < n,
	}
}

func (m Model) renderQueuedMessageWidgets(width int) string {
	blocks := make([]string, 0, 2)
	if steering := RenderQueuedMessageWidget("steering", m.steeringMessages, width); steering != "" {
		blocks = append(blocks, steering)
	}
	if queued := RenderQueuedMessageWidget("queued", m.queuedMessages, width); queued != "" {
		blocks = append(blocks, queued)
	}
	return strings.Join(blocks, "\n")
}

// RenderQueuedMessageWidget renders a width-bounded ComposerPane queue block.
// It shows at most QueueWindowSize queued drafts at a time, with stable FIFO
// numbering so operators can see what will steer or submit next.
func RenderQueuedMessageWidget(label string, q QueuedMessages, width int) string {
	items := q.Items()
	if len(items) == 0 || width <= 0 {
		return ""
	}
	label = strings.TrimSpace(label)
	if label == "" {
		label = "queued"
	}
	var editPtr *int
	if editIdx, ok := q.EditIndex(); ok {
		editPtr = &editIdx
	}
	win := ComputeQueueWindow(len(items), editPtr)
	lines := []string{fmt.Sprintf("%s (%d)", label, len(items))}
	if win.ShowLead {
		lines = append(lines, "  …")
	}
	for i := win.Start; i < win.End; i++ {
		marker := " "
		if editPtr != nil && *editPtr == i {
			marker = "▸"
		}
		line := fmt.Sprintf("%s %d. %s", marker, i+1, strings.TrimSpace(items[i]))
		lines = append(lines, line)
	}
	if win.ShowTail {
		remaining := len(items) - win.End
		lines = append(lines, fmt.Sprintf("  … and %d more", remaining))
	}
	for i, line := range lines {
		lines[i] = hermesStatusTrimToWidth(line, width)
	}
	return strings.Join(lines, "\n")
}
