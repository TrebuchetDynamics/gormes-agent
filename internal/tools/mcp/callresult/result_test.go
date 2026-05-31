package callresult

import (
	"encoding/json"
	"testing"
)

func TestParsePreservesStructuredContentEnvelope(t *testing.T) {
	raw := json.RawMessage(`{
		"content":[{"type":"text","text":"summary"}],
		"structuredContent":{"items":[{"id":"a1","score":0.75}],"nextCursor":"cursor-2"},
		"isError":false
	}`)

	got, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(got.Content) != 1 || got.Content[0].Text != "summary" {
		t.Fatalf("Content = %#v, want text summary", got.Content)
	}
	if len(got.StructuredContent) == 0 {
		t.Fatalf("StructuredContent was dropped")
	}

	var decoded struct {
		Items []struct {
			ID    string  `json:"id"`
			Score float64 `json:"score"`
		} `json:"items"`
		NextCursor string `json:"nextCursor"`
	}
	if err := json.Unmarshal(got.StructuredContent, &decoded); err != nil {
		t.Fatalf("StructuredContent is not replayable JSON: %v", err)
	}
	if decoded.NextCursor != "cursor-2" || len(decoded.Items) != 1 || decoded.Items[0].ID != "a1" {
		t.Fatalf("StructuredContent = %#v, want replayable envelope", decoded)
	}
}

func TestParseIgnoresNullStructuredContent(t *testing.T) {
	got, err := Parse(json.RawMessage(`{"content":[],"structuredContent":null}`))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if got.StructuredContent != nil {
		t.Fatalf("StructuredContent = %s, want nil for explicit null", got.StructuredContent)
	}
}
