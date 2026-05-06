package hermes

import "testing"

func TestOpenAICompatibleMessageContent_MultimodalParts(t *testing.T) {
	got := openAICompatibleMessageContent(Message{
		Role: "user",
		ContentParts: []MessageContentPart{
			{Type: "text", Text: "describe"},
			{Type: "image_url", ImageURL: "https://example.com/cat.png", Detail: "high"},
		},
	})

	parts, ok := got.([]map[string]any)
	if !ok {
		t.Fatalf("content = %T, want []map[string]any", got)
	}
	if len(parts) != 2 {
		t.Fatalf("len(parts) = %d, want 2: %#v", len(parts), parts)
	}
	if parts[0]["type"] != "text" || parts[0]["text"] != "describe" {
		t.Fatalf("text part = %#v", parts[0])
	}
	image, ok := parts[1]["image_url"].(map[string]any)
	if !ok {
		t.Fatalf("image_url = %T, want map[string]any", parts[1]["image_url"])
	}
	if parts[1]["type"] != "image_url" || image["url"] != "https://example.com/cat.png" || image["detail"] != "high" {
		t.Fatalf("image part = %#v", parts[1])
	}
}
