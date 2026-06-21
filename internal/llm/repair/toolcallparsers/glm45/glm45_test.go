package glm45

import (
	"encoding/json"
	"testing"
)

func TestParseBlock_SingleCall(t *testing.T) {
	input := "<tool_call>\n<name>get_weather</name>\n<arguments>\n<arg_key>city</arg_key><arg_value>Paris</arg_value>\n</arguments>\n</tool_call>"
	calls, errs := ParseBlock(input)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(calls) != 1 || calls[0].Name != "get_weather" {
		t.Fatalf("expected 1 call get_weather, got %+v", calls)
	}
	var args map[string]string
	if err := json.Unmarshal([]byte(calls[0].Arguments), &args); err != nil {
		t.Fatalf("arguments not valid JSON: %v", err)
	}
	if args["city"] != "Paris" {
		t.Errorf("city = %q, want Paris", args["city"])
	}
}

func TestParseBlock_MultipleArgs(t *testing.T) {
	input := "<tool_call><name>fn</name><arguments><arg_key>a</arg_key><arg_value>1</arg_value><arg_key>b</arg_key><arg_value>2</arg_value></arguments></tool_call>"
	calls, errs := ParseBlock(input)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	var args map[string]string
	if err := json.Unmarshal([]byte(calls[0].Arguments), &args); err != nil {
		t.Fatalf("not valid JSON: %v", err)
	}
	if args["a"] != "1" || args["b"] != "2" {
		t.Errorf("args = %v", args)
	}
}

func TestParseBlock_MissingName_Degraded(t *testing.T) {
	input := "<tool_call><arguments><arg_key>k</arg_key><arg_value>v</arg_value></arguments></tool_call>"
	calls, errs := ParseBlock(input)
	if len(errs) == 0 {
		t.Fatal("expected error for missing name")
	}
	if len(calls) != 0 {
		t.Errorf("expected no calls, got %d", len(calls))
	}
}

func TestParseBlock_NoArgKeyValue_EmptyArgs(t *testing.T) {
	input := "<tool_call><name>noop</name><arguments></arguments></tool_call>"
	calls, errs := ParseBlock(input)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(calls) != 1 || calls[0].Arguments != "{}" {
		t.Errorf("expected {}, got %q", calls[0].Arguments)
	}
}

func TestContainsToolCallBlock(t *testing.T) {
	if !ContainsToolCallBlock("<tool_call><name>fn</name><arguments><arg_key>x</arg_key><arg_value>1</arg_value></arguments></tool_call>") {
		t.Error("expected true")
	}
	if ContainsToolCallBlock("plain text") {
		t.Error("expected false")
	}
}
