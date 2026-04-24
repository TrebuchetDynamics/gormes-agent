package tools

import (
	"context"
	"encoding/json"
	"time"
)

// MockTool is a test double. Every field is independently configurable so
// tests can script happy paths, slow executions, panics, or ctx-cancel
// scenarios. Not used in production code.
type MockTool struct {
	NameStr    string
	Desc       string
	SchemaJSON json.RawMessage
	TimeoutD   time.Duration
	// ExecuteFn drives the behaviour. If nil, Execute returns `{"ok":true}`.
	ExecuteFn func(ctx context.Context, args json.RawMessage) (json.RawMessage, error)
}

// Compile-time interface check.
var _ Tool = (*MockTool)(nil)

func (m *MockTool) Name() string {
	if m.NameStr == "" {
		return "mock"
	}
	return m.NameStr
}

func (m *MockTool) Description() string {
	if m.Desc == "" {
		return "mock tool for testing"
	}
	return m.Desc
}

func (m *MockTool) Schema() json.RawMessage {
	if len(m.SchemaJSON) == 0 {
		return json.RawMessage(`{"type":"object","properties":{},"required":[]}`)
	}
	return m.SchemaJSON
}

func (m *MockTool) Timeout() time.Duration { return m.TimeoutD }

func (m *MockTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	if m.ExecuteFn != nil {
		return m.ExecuteFn(ctx, args)
	}
	return json.RawMessage(`{"ok":true}`), nil
}
