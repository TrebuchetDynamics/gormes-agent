package queue

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/queue/buffer"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/queue/command"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/queue/widget"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/queue/window"
)

// WindowSize is the number of queued messages rendered per frame. Tracks
// hermes-agent/ui-tui/src/components/queuedMessages.tsx@ea1012f5 (QUEUE_WINDOW=3).
const WindowSize = window.Size

type SlashResult = command.SlashResult

type Messages = buffer.Messages

type Window = window.Window

type TrimFunc = widget.TrimFunc

func HandleSlash(input string, currentLen int) SlashResult {
	return command.HandleSlash(input, currentLen)
}

func ComputeWindow(n int, editIdx *int) Window {
	return window.Compute(n, editIdx)
}

// RenderWidget renders a width-bounded ComposerPane queue block. It shows at
// most WindowSize queued drafts at a time, with stable FIFO numbering so
// operators can see what will steer or submit next.
func RenderWidget(label string, q Messages, width int, trim TrimFunc) string {
	return widget.Render(label, q, width, trim)
}
