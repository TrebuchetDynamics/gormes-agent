package callresult

import (
	"bytes"
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

func TestParsePreservesResourceLinkMetadataAndEmbeddedText(t *testing.T) {
	raw := json.RawMessage(`{
		"content":[
			{"type":"resource_link","uri":"slack://files/F123","name":"report.pdf","mimeType":"application/pdf"},
			{"type":"resource","resource":{"uri":"mem://error","mimeType":"text/plain","text":"quota exceeded for workspace W1"}}
		],
		"isError":true
	}`)

	got, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if !got.IsError {
		t.Fatal("IsError = false, want true")
	}
	if len(got.Content) != 2 {
		t.Fatalf("Content = %#v, want resource link and embedded resource", got.Content)
	}
	link := got.Content[0]
	if link.Kind != "resource_link" || link.URI != "slack://files/F123" || link.Name != "report.pdf" || link.MimeType != "application/pdf" {
		t.Fatalf("resource link = %#v, want preserved type/uri/name/mimeType", link)
	}
	embedded := got.Content[1]
	if embedded.Kind != "resource" || embedded.URI != "mem://error" || embedded.MimeType != "text/plain" || embedded.Text != "quota exceeded for workspace W1" {
		t.Fatalf("embedded resource = %#v, want nested text/uri/mimeType", embedded)
	}
}

func TestParseIgnoresNullStructuredContent(t *testing.T) {
	for _, raw := range []json.RawMessage{
		nil,
		json.RawMessage(`null`),
		json.RawMessage(`   null  `),
		json.RawMessage(`{"content":[],"structuredContent":null}`),
	} {
		got, err := Parse(raw)
		if err != nil {
			t.Fatalf("Parse(%q) returned error: %v", raw, err)
		}
		if got.StructuredContent != nil {
			t.Fatalf("StructuredContent = %s, want nil for explicit nullish input %q", got.StructuredContent, raw)
		}
	}
}

func TestParseIgnoresWhitespaceOnlyResult(t *testing.T) {
	got, err := Parse(json.RawMessage(" \n\t "))
	if err != nil {
		t.Fatalf("Parse returned error for whitespace-only result: %v", err)
	}
	if len(got.Content) != 0 || got.StructuredContent != nil || got.IsError {
		t.Fatalf("Parse whitespace-only result = %#v, want empty success result", got)
	}
}

func TestParseCopiesStructuredContentEnvelope(t *testing.T) {
	raw := json.RawMessage(`{"structuredContent":{"items":[{"id":"a1"}]}}`)

	got, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(got.StructuredContent) == 0 {
		t.Fatal("StructuredContent was dropped")
	}
	idx := bytes.Index(raw, []byte(`a1`))
	if idx < 0 {
		t.Fatalf("test fixture missing mutable token: %s", raw)
	}
	raw[idx+1] = '2'

	if string(got.StructuredContent) != `{"items":[{"id":"a1"}]}` {
		t.Fatalf("StructuredContent aliases caller buffer: %s", got.StructuredContent)
	}
}
