package hermesxml_test

import (
	"encoding/json"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/repair/toolcallparsers/hermesxml"
)

func TestParseBlock_BasicDouble(t *testing.T) {
	text := "<tool_call>\n{\"name\": \"read_file\", \"arguments\": {\"path\": \"/tmp/x.txt\"}}\n</tool_call>"
	calls, errs := hermesxml.ParseBlock(text)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "read_file" {
		t.Errorf("expected name=read_file, got %q", calls[0].Name)
	}
	var args map[string]any
	if err := json.Unmarshal(calls[0].Arguments, &args); err != nil {
		t.Fatalf("arguments not valid JSON: %v", err)
	}
	if args["path"] != "/tmp/x.txt" {
		t.Errorf("unexpected path: %v", args["path"])
	}
}

func TestParseBlock_SingleQuotedBody(t *testing.T) {
	// Python dict style — Hermes models sometimes emit this.
	text := "<tool_call>\n{'name': 'bash', 'arguments': {'cmd': 'ls -la'}}\n</tool_call>"
	calls, errs := hermesxml.ParseBlock(text)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(calls) != 1 || calls[0].Name != "bash" {
		t.Fatalf("expected bash call, got %+v errs=%v", calls, errs)
	}
}

func TestParseBlock_MultipleBlocks(t *testing.T) {
	text := `Thinking...
<tool_call>
{"name": "read_file", "arguments": {"path": "a.txt"}}
</tool_call>
Some text.
<tool_call>
{"name": "write_file", "arguments": {"path": "b.txt", "content": "hi"}}
</tool_call>
Done.`
	calls, errs := hermesxml.ParseBlock(text)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}
	if calls[0].Name != "read_file" || calls[1].Name != "write_file" {
		t.Errorf("unexpected names: %q %q", calls[0].Name, calls[1].Name)
	}
}

func TestParseBlock_EmptyArguments(t *testing.T) {
	text := "<tool_call>\n{\"name\": \"list_files\", \"arguments\": {}}\n</tool_call>"
	calls, errs := hermesxml.ParseBlock(text)
	if len(errs) != 0 || len(calls) != 1 {
		t.Fatalf("expected 1 call, got calls=%v errs=%v", calls, errs)
	}
	if string(calls[0].Arguments) != "{}" {
		t.Errorf("unexpected arguments: %s", calls[0].Arguments)
	}
}

func TestParseBlock_NullArguments(t *testing.T) {
	text := "<tool_call>\n{\"name\": \"noop\", \"arguments\": null}\n</tool_call>"
	calls, errs := hermesxml.ParseBlock(text)
	if len(errs) != 0 || len(calls) != 1 {
		t.Fatalf("expected 1 call, got calls=%v errs=%v", calls, errs)
	}
	// null arguments should normalise to {}
	if string(calls[0].Arguments) != "{}" {
		t.Errorf("expected {}, got %s", calls[0].Arguments)
	}
}

func TestParseBlock_MalformedJSON(t *testing.T) {
	text := "<tool_call>\nnot json at all\n</tool_call>"
	calls, errs := hermesxml.ParseBlock(text)
	if len(calls) != 0 {
		t.Errorf("expected no calls from malformed JSON, got %d", len(calls))
	}
	if len(errs) != 1 {
		t.Errorf("expected 1 error from malformed block, got %d", len(errs))
	}
}

func TestParseBlock_UnclosedTag(t *testing.T) {
	text := "<tool_call>\n{\"name\": \"foo\", \"arguments\": {}}\n"
	calls, errs := hermesxml.ParseBlock(text)
	if len(calls) != 0 {
		t.Errorf("expected no calls from unclosed tag, got %d", len(calls))
	}
	if len(errs) == 0 {
		t.Error("expected an error for unclosed tag")
	}
}

func TestParseBlock_NoToolCall(t *testing.T) {
	text := "Hello, this is a normal response with no tool calls."
	calls, errs := hermesxml.ParseBlock(text)
	if len(calls) != 0 || len(errs) != 0 {
		t.Errorf("expected empty result, got calls=%v errs=%v", calls, errs)
	}
}

func TestParseBlock_MissingName(t *testing.T) {
	text := "<tool_call>\n{\"arguments\": {\"x\": 1}}\n</tool_call>"
	calls, errs := hermesxml.ParseBlock(text)
	if len(calls) != 0 {
		t.Errorf("expected no calls when name is empty")
	}
	if len(errs) == 0 {
		t.Error("expected error for missing name")
	}
}

func TestContainsToolCallBlock(t *testing.T) {
	if !hermesxml.ContainsToolCallBlock("text <tool_call> stuff") {
		t.Error("expected true when <tool_call> present")
	}
	if hermesxml.ContainsToolCallBlock("no tool calls here") {
		t.Error("expected false when <tool_call> absent")
	}
}
