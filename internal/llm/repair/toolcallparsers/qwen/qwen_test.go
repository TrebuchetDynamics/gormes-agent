package qwen_test

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/repair/toolcallparsers/qwen"
)

func TestQwenParseBlock_Basic(t *testing.T) {
	text := "<tool_call>\n{\"name\": \"search\", \"arguments\": {\"query\": \"golang\"}}\n</tool_call>"
	calls, errs := qwen.ParseBlock(text)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(calls) != 1 || calls[0].Name != "search" {
		t.Errorf("expected search call, got %+v", calls)
	}
}

func TestQwenContainsToolCallBlock(t *testing.T) {
	if !qwen.ContainsToolCallBlock("text <tool_call> content") {
		t.Error("expected true")
	}
	if qwen.ContainsToolCallBlock("no tool calls") {
		t.Error("expected false")
	}
}
