package deepseekv3

import "testing"

func TestParseBlock_WithCodeFence(t *testing.T) {
	input := "<｜tool▁calls▁begin｜><｜tool▁call▁begin｜>get_weather<｜tool▁sep｜>```json\n{\"city\": \"Paris\"}\n```<｜tool▁call▁end｜>"
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

func TestParseBlock_WithoutCodeFence(t *testing.T) {
	input := "<｜tool▁call▁begin｜>fn<｜tool▁sep｜>{\"x\":1}<｜tool▁call▁end｜>"
	calls, errs := ParseBlock(input)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(calls) != 1 || calls[0].Name != "fn" {
		t.Errorf("expected call fn, got %+v", calls)
	}
}

func TestParseBlock_Degraded_MissingSep(t *testing.T) {
	input := "<｜tool▁call▁begin｜>fn<｜tool▁call▁end｜>"
	_, errs := ParseBlock(input)
	if len(errs) == 0 {
		t.Fatal("expected degraded error for missing separator")
	}
}

func TestContainsToolCallBlock(t *testing.T) {
	if !ContainsToolCallBlock("<｜tool▁call▁begin｜>fn<｜tool▁sep｜>{}") {
		t.Error("expected true")
	}
	if ContainsToolCallBlock("plain text") {
		t.Error("expected false")
	}
}
