package longcat

import "testing"

func TestParseBlock_Delegates_To_HermesXML(t *testing.T) {
	input := `<tool_call>{"name": "fn", "arguments": {"x": 1}}</tool_call>`
	calls, errs := ParseBlock(input)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(calls) != 1 || calls[0].Name != "fn" {
		t.Fatalf("expected 1 call fn, got %+v", calls)
	}
}

func TestContainsToolCallBlock(t *testing.T) {
	if !ContainsToolCallBlock("<tool_call>{}") {
		t.Error("expected true")
	}
	if ContainsToolCallBlock("plain text") {
		t.Error("expected false")
	}
}
