package interrupt

import (
	"encoding/json"
	"testing"
)

func TestInterruptTool_Schema(t *testing.T) {
	tool := NewTool(nil)
	if tool.Name() != "interrupt" {
		t.Fatalf("Name() = %q", tool.Name())
	}
}

func TestInterruptTool_Execute(t *testing.T) {
	called := false
	tool := NewTool(func() { called = true })
	result, err := tool.Execute(nil, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !called {
		t.Fatal("interruptFn not called")
	}
	var out map[string]any
	json.Unmarshal(result, &out)
	if out["success"] != true {
		t.Fatal("expected success")
	}
}

func TestInterruptTool_NilCallback(t *testing.T) {
	tool := NewTool(nil)
	_, err := tool.Execute(nil, nil)
	if err != nil {
		t.Fatalf("Execute with nil callback: %v", err)
	}
}
