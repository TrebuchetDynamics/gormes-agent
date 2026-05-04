package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestTransactionalExecutor_SafeCommand(t *testing.T) {
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

func TestTransactionalExecutor_UnsafeBlocked(t *testing.T) {
	cc := NewCommandClassifier()
	te := NewTransactionalExecutor(nil, cc)

	result, err := te.Execute(context.Background(), ToolRequest{
		ToolName: "terminal",
		Input:    json.RawMessage(`sudo rm -rf /`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Success {
		t.Fatal("unsafe command should be blocked")
	}
	if result.Classification != "unsafe" {
		t.Fatalf("classification = %s, want unsafe", result.Classification)
	}
}

func TestTransactionalExecutor_UncertainRunsWithRollback(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&NowTool{})
	inner := NewInProcessToolExecutor(reg)
	te := NewTransactionalExecutor(inner, NewCommandClassifier())

	result, err := te.Execute(context.Background(), ToolRequest{
		ToolName: "now",
		Input:    json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success {
		t.Fatalf("now tool should succeed, got: %s", result.Error)
	}
	if !result.SnapshotTaken {
		t.Fatalf("uncertain command SnapshotTaken = false, want snapshot wrapper before execution")
	}
}
