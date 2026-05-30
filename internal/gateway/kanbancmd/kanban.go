package kanbancmd

import (
	"context"
	"errors"
	"strings"
)

const MaxOutputBytes = 3800

// Runner executes a complete /kanban slash command line and returns channel
// text output.
type Runner func(context.Context, string) (string, error)

// RunSlash normalizes gateway /kanban input before invoking the injected
// runner. Empty input maps to /kanban so channels show the same help surface as
// terminal invocations.
func RunSlash(ctx context.Context, runner Runner, input string) (string, error) {
	if runner == nil {
		return "", errors.New("kanban command runner unavailable")
	}
	input = strings.TrimSpace(input)
	if input == "" {
		input = "/kanban"
	}
	return runner(ctx, input)
}

// BoundOutput trims empty and oversized runner output for channel delivery.
func BoundOutput(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return "(no output)"
	}
	if len(output) <= MaxOutputBytes {
		return output
	}
	return output[:MaxOutputBytes] + "\n... (truncated; use `gormes kanban ...` in your terminal for full output)"
}
