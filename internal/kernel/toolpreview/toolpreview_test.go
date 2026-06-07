package toolpreview

import (
	"encoding/json"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestSoulTextUsesSharedPreviewFormatting(t *testing.T) {
	args := json.RawMessage(`{"action":"send", "session_id":"1234567890abcdefZZ", "data":"hello   world from preview"}`)
	got := SoulText(llm.ToolCall{Name: " process ", Arguments: args})
	want := `tool: process: send 1234567890abcdef "hello world from pre"`
	if got != want {
		t.Fatalf("SoulText = %q, want %q", got, want)
	}
}

func TestPreviewUsesToolSpecificAndFallbackArguments(t *testing.T) {
	if got := Preview("todo", json.RawMessage(`{"merge":true,"todos":[{},{}]}`)); got != "updating 2 task(s)" {
		t.Fatalf("todo preview = %q", got)
	}
	if got := Preview("browser_click", json.RawMessage(`{"ref":"button-1"}`)); got != "button-1" {
		t.Fatalf("browser_click preview = %q", got)
	}
	if got := Preview("custom", json.RawMessage(`{"query":"  hi   there  "}`)); got != "hi there" {
		t.Fatalf("fallback preview = %q", got)
	}
}

func TestPreviewHandlesInvalidOrUnknownArguments(t *testing.T) {
	if got := Preview("terminal", json.RawMessage(`not-json`)); got != "" {
		t.Fatalf("invalid preview = %q", got)
	}
	if got := SoulText(llm.ToolCall{}); got != "tool: unknown" {
		t.Fatalf("unknown soul text = %q", got)
	}
	if got := TruncateToken("abcdef", 3); got != "abc" {
		t.Fatalf("TruncateToken = %q", got)
	}
}
