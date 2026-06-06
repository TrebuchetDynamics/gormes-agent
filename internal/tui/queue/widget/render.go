package widget

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/queue/buffer"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/queue/window"
)

// TrimFunc trims a rendered queue line to the caller's display width policy.
type TrimFunc func(string, int) string

// Render renders a width-bounded ComposerPane queue block. It shows at
// most window.Size queued drafts at a time, with stable FIFO numbering so
// operators can see what will steer or submit next.
func Render(label string, q buffer.Messages, width int, trim TrimFunc) string {
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
	win := window.Compute(len(items), editPtr)
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
	if trim != nil {
		for i, line := range lines {
			lines[i] = trim(line, width)
		}
	}
	return strings.Join(lines, "\n")
}
