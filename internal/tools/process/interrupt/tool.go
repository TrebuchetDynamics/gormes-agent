package interrupt

import (
	"context"
	"encoding/json"
	"time"
)

type Tool struct {
	interruptFn func()
}

func NewTool(fn func()) *Tool {
	return &Tool{interruptFn: fn}
}

func (*Tool) Name() string { return "interrupt" }
func (*Tool) Description() string {
	return "Request interruption of the current agent turn. Use when the user asks to stop."
}
func (*Tool) Timeout() time.Duration { return 2 * time.Second }

func (*Tool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{},"description":"Interrupt the current agent execution. No parameters needed."}`)
}

func (t *Tool) Execute(_ context.Context, args json.RawMessage) (json.RawMessage, error) {
	if t.interruptFn != nil {
		t.interruptFn()
	}
	return json.Marshal(map[string]any{
		"success": true,
		"message": "Interrupt signal sent. The agent will stop after the current turn.",
	})
}
