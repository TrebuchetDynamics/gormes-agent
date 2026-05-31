package jsonc

import (
	"encoding/json"
	"testing"
)

func TestStripJSONCommentsPreservesStringContents(t *testing.T) {
	input := `[
	  // comment
	  {"key":"a","args":{"text":"// not a comment",},},
	  /* block */ {"key":"b"},
	]`
	var parsed []map[string]any
	if err := json.Unmarshal([]byte(StripJSONComments(input)), &parsed); err != nil {
		t.Fatalf("StripJSONComments output did not parse: %v", err)
	}
	if parsed[0]["key"] != "a" || parsed[0]["args"].(map[string]any)["text"] != "// not a comment" {
		t.Fatalf("parsed = %#v, want string contents preserved", parsed)
	}
}

func TestParseKeybindingsTreatsEmptyJSONCAsEmptyList(t *testing.T) {
	bindings, err := ParseKeybindings([]byte(" // comment\n"))
	if err != nil {
		t.Fatalf("ParseKeybindings() error = %v", err)
	}
	if len(bindings) != 0 {
		t.Fatalf("ParseKeybindings() = %#v, want empty list", bindings)
	}
}
