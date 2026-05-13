package tools

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit"
)

func TestToolkitFacadeContractsStayAliased(t *testing.T) {
	reg := NewRegistry()
	var toolkitRegistry *toolkit.Registry = reg
	toolkitRegistry.MustRegister(facadeNoopTool{})

	descriptors := reg.Descriptors()
	if len(descriptors) != 1 {
		t.Fatalf("expected one descriptor, got %d", len(descriptors))
	}

	var toolkitDescriptor toolkit.ToolDescriptor = descriptors[0]
	var facadeDescriptor ToolDescriptor = toolkitDescriptor
	if facadeDescriptor.Name != "facade_noop" {
		t.Fatalf("unexpected descriptor name %q", facadeDescriptor.Name)
	}

	executor := NewInProcessToolExecutor(reg)
	var toolkitExecutor *toolkit.InProcessToolExecutor = executor
	events, err := toolkitExecutor.Execute(context.Background(), toolkit.ToolRequest{
		ToolName: "facade_noop",
		Input:    json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	var last toolkit.ToolEvent
	for event := range events {
		var facadeEvent ToolEvent = event
		last = facadeEvent
	}
	if last.Type != "completed" {
		t.Fatalf("expected completed event, got %q", last.Type)
	}
}

type facadeNoopTool struct{}

func (facadeNoopTool) Name() string            { return "facade_noop" }
func (facadeNoopTool) Description() string     { return "test tool" }
func (facadeNoopTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (facadeNoopTool) Timeout() time.Duration  { return time.Second }
func (facadeNoopTool) Execute(context.Context, json.RawMessage) (json.RawMessage, error) {
	return json.RawMessage(`{"ok":true}`), nil
}
