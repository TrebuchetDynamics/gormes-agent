package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestMockTool_DefaultExecute(t *testing.T) {
	m := &MockTool{}
	out, err := m.Execute(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `{"ok":true}` {
		t.Errorf("default Execute = %s", out)
	}
}

func TestMockTool_CustomExecute(t *testing.T) {
	m := &MockTool{
		ExecuteFn: func(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
			return nil, errors.New("boom")
		},
	}
	_, err := m.Execute(context.Background(), nil)
	if err == nil || err.Error() != "boom" {
		t.Errorf("err = %v, want boom", err)
	}
}

func TestMockTool_DefaultsForMissingFields(t *testing.T) {
	m := &MockTool{}
	if m.Name() != "mock" {
		t.Errorf("default name = %q", m.Name())
	}
	if m.Description() == "" {
		t.Error("default description should be non-empty")
	}
	var schema map[string]any
	if err := json.Unmarshal(m.Schema(), &schema); err != nil {
		t.Errorf("default schema invalid: %v", err)
	}
}
