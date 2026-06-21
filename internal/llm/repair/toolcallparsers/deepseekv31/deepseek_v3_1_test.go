package deepseekv31

import "testing"

func TestParseBlock_SingleCall(t *testing.T) {
	input := "header <｜tool▁calls▁begin｜><｜tool▁call▁begin｜>get_weather<｜tool▁sep｜>{\"city\": \"Paris\"}<｜tool▁call▁end｜>"
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
	if calls[0].Arguments != `{"city": "Paris"}` {
		t.Errorf("arguments = %q, want %q", calls[0].Arguments, `{"city": "Paris"}`)
	}
}

func TestParseBlock_MultipleCall(t *testing.T) {
	input := "<｜tool▁calls▁begin｜><｜tool▁call▁begin｜>foo<｜tool▁sep｜>{\"a\":1}<｜tool▁call▁end｜><｜tool▁call▁begin｜>bar<｜tool▁sep｜>{\"b\":2}<｜tool▁call▁end｜>"
	calls, errs := ParseBlock(input)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d: %+v", len(calls), calls)
	}
	if calls[0].Name != "foo" || calls[1].Name != "bar" {
		t.Errorf("names = %q, %q; want foo, bar", calls[0].Name, calls[1].Name)
	}
}

func TestParseBlock_MissingSep_Degraded(t *testing.T) {
	input := "<｜tool▁call▁begin｜>oops<｜tool▁call▁end｜>"
	calls, errs := ParseBlock(input)
	if len(errs) == 0 {
		t.Fatal("expected at least one error for missing separator")
	}
	if len(calls) != 0 {
		t.Errorf("expected no calls on degraded input, got %d", len(calls))
	}
}

func TestParseBlock_InvalidJSON_Degraded(t *testing.T) {
	input := "<｜tool▁call▁begin｜>fn<｜tool▁sep｜>NOT_JSON<｜tool▁call▁end｜>"
	calls, errs := ParseBlock(input)
	if len(errs) == 0 {
		t.Fatal("expected degraded error for invalid JSON")
	}
	if len(calls) != 1 || calls[0].Name != "fn" {
		t.Errorf("expected degraded call with name fn, got %+v", calls)
	}
	if calls[0].Arguments != "{}" {
		t.Errorf("degraded arguments = %q, want {}", calls[0].Arguments)
	}
}

func TestContainsToolCallBlock(t *testing.T) {
	if !ContainsToolCallBlock("<｜tool▁call▁begin｜>foo<｜tool▁sep｜>{}") {
		t.Error("expected true for valid tool call block")
	}
	if ContainsToolCallBlock("plain text") {
		t.Error("expected false for plain text")
	}
}
