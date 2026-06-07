package toolpayload

import (
	"encoding/json"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestMultimodalResultParsesTextAndImageParts(t *testing.T) {
	payload := json.RawMessage(`{
		"_multimodal": true,
		"content": [
			{"type":"output_text","text":" hello "},
			{"type":"input_image","image_url":{"url":"https://example.test/a.png","detail":"high"}}
		]
	}`)
	summary, parts, ok := MultimodalResult(payload)
	if !ok {
		t.Fatal("MultimodalResult ok = false, want true")
	}
	if summary != "hello" {
		t.Fatalf("summary = %q, want hello", summary)
	}
	if len(parts) != 2 || parts[1].Type != "image_url" || parts[1].ImageURL == "" || parts[1].Detail != "high" {
		t.Fatalf("parts = %#v, want text and detailed image_url", parts)
	}
}

func TestMultimodalResultUsesExplicitSummaryAndRejectsEmpty(t *testing.T) {
	payload := json.RawMessage(`{"_multimodal": true, "text_summary":"explicit", "content":[{"type":"image_url","url":"https://example.test/a.png"}]}`)
	summary, _, ok := MultimodalResult(payload)
	if !ok || summary != "explicit" {
		t.Fatalf("MultimodalResult = (%q, ok=%v), want explicit true", summary, ok)
	}
	if _, _, ok := MultimodalResult(json.RawMessage(`{"_multimodal": true, "content":[{"type":"text","text":"   "}]}`)); ok {
		t.Fatal("MultimodalResult accepted empty content")
	}
}

func TestAppendSubdirectoryHintToContentPartsCopiesAndAppendsToText(t *testing.T) {
	parts := []llm.MessageContentPart{{Type: "text", Text: "base"}}
	got := AppendSubdirectoryHintToContentParts(parts, " hint")
	if got[0].Text != "base hint" {
		t.Fatalf("hinted text = %q", got[0].Text)
	}
	if parts[0].Text != "base" {
		t.Fatalf("input mutated to %q", parts[0].Text)
	}
	got = AppendSubdirectoryHintToContentParts([]llm.MessageContentPart{{Type: "image_url", ImageURL: "x"}}, "hint")
	if got[0].Type != "text" || got[0].Text != "hint" || got[1].Type != "image_url" {
		t.Fatalf("prepend text hint result = %#v", got)
	}
}
