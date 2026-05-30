package moa

import (
	"encoding/json"
	"testing"
)

func TestMoATool_Name(t *testing.T) {
	tool := &MoATool{}
	if tool.Name() != "mixture_of_agents" {
		t.Fatalf("Name() = %q", tool.Name())
	}
}

func TestMoATool_Schema(t *testing.T) {
	tool := &MoATool{}
	schema := tool.Schema()
	if schema == nil {
		t.Fatal("Schema() is nil")
	}
	var s map[string]any
	if err := json.Unmarshal(schema, &s); err != nil {
		t.Fatalf("invalid schema: %v", err)
	}
	props, ok := s["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema missing properties")
	}
	if _, ok := props["prompt"]; !ok {
		t.Fatal("schema missing prompt property")
	}
	req, _ := s["required"].([]any)
	found := false
	for _, r := range req {
		if r.(string) == "prompt" {
			found = true
		}
	}
	if !found {
		t.Fatal("prompt not in required")
	}
}

func TestMoATool_Timeout(t *testing.T) {
	tool := &MoATool{}
	if tool.Timeout() <= 0 {
		t.Fatal("Timeout() should be positive")
	}
}

func TestMoATool_Execute(t *testing.T) {
	tool := &MoATool{}
	result, err := tool.Execute(nil, json.RawMessage(`{"prompt":"test"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var out struct {
		Success         bool   `json:"success"`
		Result          string `json:"result"`
		ModelsRequested int    `json:"models_requested"`
		Aggregation     string `json:"aggregation"`
		Stub            bool   `json:"stub"`
	}
	if err := json.Unmarshal(result, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !out.Success || !out.Stub {
		t.Fatal("expected success and stub")
	}
}

func TestMoATool_ExecuteWithModels(t *testing.T) {
	tool := &MoATool{}
	result, _ := tool.Execute(nil, json.RawMessage(`{"prompt":"test","models":["gpt-4","claude"]}`))
	var out struct {
		ModelsRequested int  `json:"models_requested"`
		Stub            bool `json:"stub"`
	}
	json.Unmarshal(result, &out)
	if out.ModelsRequested != 2 {
		t.Fatalf("models_requested = %d, want 2", out.ModelsRequested)
	}
	if !out.Stub {
		t.Fatal("expected stub")
	}
}

func TestMoATool_Description(t *testing.T) {
	tool := &MoATool{}
	if tool.Description() == "" {
		t.Fatal("Description() is empty")
	}
}
