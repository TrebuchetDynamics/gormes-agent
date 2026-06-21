package llama

import "testing"

func TestParseBlock_WithPythonTag(t *testing.T) {
	input := `<|python_tag|>{"name": "get_weather", "parameters": {"city": "Paris"}}`
	calls, errs := ParseBlock(input)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(calls) != 1 || calls[0].Name != "get_weather" {
		t.Fatalf("expected 1 call get_weather, got %+v", calls)
	}
	if calls[0].Arguments != `{"city": "Paris"}` {
		t.Errorf("arguments = %q", calls[0].Arguments)
	}
}

func TestParseBlock_BareJSON(t *testing.T) {
	input := `{"name": "search", "arguments": {"query": "Go generics"}}`
	calls, errs := ParseBlock(input)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(calls) != 1 || calls[0].Name != "search" {
		t.Fatalf("expected 1 call search, got %+v", calls)
	}
}

func TestParseBlock_Preamble(t *testing.T) {
	input := "thinking...\n<|python_tag|>{\"name\": \"fn\", \"parameters\": {}}"
	calls, errs := ParseBlock(input)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(calls) != 1 || calls[0].Name != "fn" {
		t.Errorf("expected call fn, got %+v", calls)
	}
}

func TestParseBlock_MalformedJSON_Degraded(t *testing.T) {
	input := "<|python_tag|>NOT_JSON"
	calls, errs := ParseBlock(input)
	if len(errs) == 0 {
		t.Fatal("expected degraded error for malformed JSON")
	}
	if len(calls) != 0 {
		t.Errorf("expected no calls, got %d", len(calls))
	}
}

func TestParseBlock_MissingName_Degraded(t *testing.T) {
	input := `{"parameters": {"x": 1}}`
	_, errs := ParseBlock(input)
	if len(errs) == 0 {
		t.Fatal("expected error for missing name")
	}
}

func TestContainsToolCallBlock(t *testing.T) {
	if !ContainsToolCallBlock("<|python_tag|>{}") {
		t.Error("expected true for python_tag")
	}
	if !ContainsToolCallBlock(`{"name": "foo"}`) {
		t.Error("expected true for bare JSON with name")
	}
	if ContainsToolCallBlock("plain text") {
		t.Error("expected false for plain text")
	}
}
