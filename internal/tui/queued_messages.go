package tui

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/queue"
)

// QueueWindowSize is the number of queued messages rendered per frame. Tracks
// hermes-agent/ui-tui/src/components/queuedMessages.tsx@ea1012f5 (QUEUE_WINDOW=3).
const QueueWindowSize = queue.WindowSize

// QueuedMessages is a pure helper that owns the buffer of pending operator
// turns plus the optional edit selection. It is deliberately isolated from
// Bubble Tea wiring: callers manipulate the buffer directly, then read Items
// and ComputeQueueWindow when rendering. Tracks Hermes useQueue from
// ui-tui/src/hooks/useQueue.ts@ea1012f5.
type QueuedMessages = queue.Messages

// QueueWindow describes the visible slice of the queued buffer for rendering.
// Start is inclusive, End is exclusive. ShowLead/ShowTail signal the operator
// that hidden items exist before/after the window so the caller can render
// the leading "…" and trailing "…and N more" affordances. Tracks Hermes'
// getQueueWindow shape exactly.
type QueueWindow = queue.Window

// ComputeQueueWindow returns the three-row visible window for a queue of
// length n. When editIdx is nil the window anchors at the head; otherwise
// the window centres around the edited item but never advances past
// max(0, n-QueueWindowSize). Mirrors the start/end/showLead/showTail formula
// from hermes-agent/ui-tui/src/components/queuedMessages.tsx@ea1012f5.
func ComputeQueueWindow(n int, editIdx *int) QueueWindow {
	return queue.ComputeWindow(n, editIdx)
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
