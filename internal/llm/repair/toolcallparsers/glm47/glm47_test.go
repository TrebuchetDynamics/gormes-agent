package glm47

import "testing"

func TestParseBlock_NewlineToleranceAroundTags(t *testing.T) {
	input := "<tool_call>\n  <name>\n  get_weather\n  </name>\n  <arguments>\n    <arg_key>\n    city\n    </arg_key>\n    <arg_value>\n    Paris\n    </arg_value>\n  </arguments>\n</tool_call>"
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

func TestContainsToolCallBlock(t *testing.T) {
	if !ContainsToolCallBlock("<tool_call><name>fn</name><arguments><arg_key>k</arg_key><arg_value>v</arg_value></arguments></tool_call>") {
		t.Error("expected true")
	}
}
