package prompts

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEncodePromptContext_OptionalTOONFormat(t *testing.T) {
	raw := json.RawMessage(`{"rows":[{"id":1,"name":"Ada"},{"id":2,"name":"Bob"}],"source":"fixture"}`)

	encoded, report, err := EncodePromptContext(raw, PromptContextFormatTOON)
	if err != nil {
		t.Fatalf("EncodePromptContext TOON: %v", err)
	}
	if report.Format != string(PromptContextFormatTOON) {
		t.Fatalf("report.Format = %q, want toon", report.Format)
	}
	if !strings.Contains(string(encoded), "rows[2]{id,name}:") {
		t.Fatalf("TOON output missing tabular rows:\n%s", encoded)
	}
	if report.EncodedBytes >= report.RawBytes {
		t.Fatalf("TOON should be smaller for tabular context: raw=%d encoded=%d output=%s", report.RawBytes, report.EncodedBytes, encoded)
	}
}

func TestEncodePromptContext_DefaultFormatIsTOON(t *testing.T) {
	raw := json.RawMessage(`{"rows":[{"id":1,"name":"Ada"},{"id":2,"name":"Bob"}],"source":"fixture"}`)

	encoded, report, err := EncodePromptContext(raw, "")
	if err != nil {
		t.Fatalf("EncodePromptContext default: %v", err)
	}
	if report.Format != string(PromptContextFormatTOON) {
		t.Fatalf("report.Format = %q, want toon", report.Format)
	}
	if !strings.Contains(string(encoded), "rows[2]{id,name}:") {
		t.Fatalf("default output should be TOON tabular context:\n%s", encoded)
	}
}

func TestEncodePromptContext_JSONRemainsDefaultAPIShape(t *testing.T) {
	raw := json.RawMessage(`{
		"rows": [
			{"id": 1, "name": "Ada"}
		]
	}`)

	encoded, report, err := EncodePromptContext(raw, PromptContextFormatJSON)
	if err != nil {
		t.Fatalf("EncodePromptContext JSON: %v", err)
	}
	if report.Format != string(PromptContextFormatJSON) {
		t.Fatalf("report.Format = %q, want json", report.Format)
	}
	if string(encoded) != `{"rows":[{"id":1,"name":"Ada"}]}` {
		t.Fatalf("JSON context should be compact JSON, got %s", encoded)
	}
}
