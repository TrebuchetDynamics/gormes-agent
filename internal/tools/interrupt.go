package tools

import (
	"context"
	"encoding/json"
	"time"
)

type InterruptTool struct {
	interruptFn func()
}

func NewInterruptTool(fn func()) *InterruptTool {
	return &InterruptTool{interruptFn: fn}
}

func (*InterruptTool) Name() string { return "interrupt" }
func (*InterruptTool) Description() string {
	return "Request interruption of the current agent turn. Use when the user asks to stop."
}
func (*InterruptTool) Timeout() time.Duration { return 2 * time.Second }

func (*InterruptTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{},"description":"Interrupt the current agent execution. No parameters needed."}`)
}

func (t *InterruptTool) Execute(_ context.Context, args json.RawMessage) (json.RawMessage, error) {
	if t.interruptFn != nil {
		t.interruptFn()
	}
	return json.Marshal(map[string]any{
		"success": true,
		"message": "Interrupt signal sent. The agent will stop after the current turn.",
	})
}
