package kimik2

import "testing"

func TestParseBlock_SingleCall(t *testing.T) {
	input := "<|tool_calls_section_begin|>\n[{\"name\": \"get_weather\", \"arguments\": {\"city\": \"Paris\"}}]\n<|tool_calls_section_end|>"
	calls, errs := ParseBlock(input)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(calls) != 1 || calls[0].Name != "get_weather" {
		t.Fatalf("expected 1 call get_weather, got %+v", calls)
	}
}

func TestParseBlock_MultipleCall(t *testing.T) {
	input := "<|tool_calls_section_begin|>[{\"name\": \"foo\", \"arguments\": {}}, {\"name\": \"bar\", \"arguments\": {\"x\": 1}}]<|tool_calls_section_end|>"
	calls, errs := ParseBlock(input)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d: %+v", len(calls), calls)
	}
}

func TestParseBlock_WithPreamble(t *testing.T) {
	input := "Some text <|tool_calls_section_begin|>[{\"name\": \"fn\", \"arguments\": {}}]<|tool_calls_section_end|>"
	calls, errs := ParseBlock(input)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(calls) != 1 || calls[0].Name != "fn" {
		t.Errorf("expected call fn, got %+v", calls)
	}
}

func TestParseBlock_Malformed_Degraded(t *testing.T) {
	input := "<|tool_calls_section_begin|>NOT_JSON<|tool_calls_section_end|>"
	calls, errs := ParseBlock(input)
	if len(errs) == 0 {
		t.Fatal("expected degraded error")
	}
	if len(calls) != 0 {
		t.Errorf("expected no calls, got %d", len(calls))
	}
}

func TestContainsToolCallBlock(t *testing.T) {
	if !ContainsToolCallBlock("<|tool_calls_section_begin|>[]") {
		t.Error("expected true")
	}
	if ContainsToolCallBlock("plain text") {
		t.Error("expected false")
	}
}
