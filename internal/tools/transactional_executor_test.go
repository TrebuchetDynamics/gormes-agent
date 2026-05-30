package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestTransactionalExecutorFacade_SafeCommand(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&EchoTool{})
	inner := NewInProcessToolExecutor(reg)
	te := NewTransactionalExecutor(inner, NewCommandClassifier())

	result, err := te.Execute(context.Background(), ToolRequest{
		ToolName: "echo",
		Input:    json.RawMessage(`{"text":"hello"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("safe echo should succeed, got: %s", result.Error)
	}
}
