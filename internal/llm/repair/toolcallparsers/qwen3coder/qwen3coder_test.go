package qwen3coder

import (
	"encoding/json"
	"testing"
)

func TestParseBlock_SingleCall(t *testing.T) {
	input := "<function=get_weather>\n<parameter=city>Paris</parameter>\n</function>"
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

func TestParseBlock_MultipleParams(t *testing.T) {
	input := "<function=fn><parameter=a>1</parameter><parameter=b>2</parameter></function>"
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

func TestParseBlock_NoParams_EmptyArgs(t *testing.T) {
	input := "<function=noop></function>"
	calls, errs := ParseBlock(input)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(calls) != 1 || calls[0].Arguments != "{}" {
		t.Errorf("expected {}, got %q; calls=%+v", calls[0].Arguments, calls)
	}
}

func TestContainsToolCallBlock(t *testing.T) {
	if !ContainsToolCallBlock("<function=fn><parameter=x>1</parameter></function>") {
		t.Error("expected true")
	}
	if ContainsToolCallBlock("plain text") {
		t.Error("expected false")
	}
}
