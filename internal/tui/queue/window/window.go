package window

// Size is the number of queued messages rendered per frame. Tracks
// hermes-agent/ui-tui/src/components/queuedMessages.tsx@ea1012f5 (QUEUE_WINDOW=3).
const Size = 3

// Window describes the visible slice of the queued buffer for rendering.
// Start is inclusive, End is exclusive. ShowLead/ShowTail signal the operator
// that hidden items exist before/after the window so the caller can render
// the leading "…" and trailing "…and N more" affordances. Tracks Hermes'
// getQueueWindow shape exactly.
type Window struct {
	Start    int
	End      int
	ShowLead bool
	ShowTail bool
}

// Compute returns the three-row visible window for a queue of
// length n. When editIdx is nil the window anchors at the head; otherwise
// the window centres around the edited item but never advances past
// max(0, n-Size). Mirrors the start/end/showLead/showTail formula
// from hermes-agent/ui-tui/src/components/queuedMessages.tsx@ea1012f5.
func Compute(n int, editIdx *int) Window {
	if n <= 0 {
		return Window{}
	}

	maxStart := n - Size
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

	end := start + Size
	if end > n {
		end = n
	}

	return Window{
		Start:    start,
		End:      end,
		ShowLead: start > 0,
		ShowTail: end < n,
	}
}
