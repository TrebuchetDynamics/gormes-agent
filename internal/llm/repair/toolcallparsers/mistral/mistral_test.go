package mistral

import "testing"

func TestParseBlock_SingleCall(t *testing.T) {
	input := `[TOOL_CALLS] [{"name": "get_weather", "arguments": {"city": "Paris"}}]`
	calls, errs := ParseBlock(input)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d: %+v", len(calls), calls)
	}
	if calls[0].Name != "get_weather" {
		t.Errorf("name = %q, want get_weather", calls[0].Name)
	}
}

func TestParseBlock_MultipleCall(t *testing.T) {
	input := `[TOOL_CALLS] [{"name": "foo", "arguments": {}}, {"name": "bar", "arguments": {"x": 1}}]`
	calls, errs := ParseBlock(input)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}
}

func TestParseBlock_WithPreamble(t *testing.T) {
	input := `Some text here [TOOL_CALLS] [{"name": "fn", "arguments": {}}]`
	calls, errs := ParseBlock(input)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(calls) != 1 || calls[0].Name != "fn" {
		t.Errorf("expected 1 call fn, got %+v", calls)
	}
}

func TestParseBlock_MalformedJSON_Degraded(t *testing.T) {
	input := `[TOOL_CALLS] NOT_JSON`
	calls, errs := ParseBlock(input)
	if len(errs) == 0 {
		t.Fatal("expected degraded error for malformed JSON")
	}
	if len(calls) != 0 {
		t.Errorf("expected no calls on malformed input, got %d", len(calls))
	}
}

func TestParseBlock_EmptySentinel_Degraded(t *testing.T) {
	input := `[TOOL_CALLS] `
	calls, errs := ParseBlock(input)
	if len(errs) == 0 {
		t.Fatal("expected degraded error for empty payload")
	}
	_ = calls
}

func TestContainsToolCallBlock(t *testing.T) {
	if !ContainsToolCallBlock("[TOOL_CALLS] [{}]") {
		t.Error("expected true")
	}
	if ContainsToolCallBlock("plain text") {
		t.Error("expected false")
	}
}
