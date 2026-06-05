package kernel

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestExecuteToolCalls_MultimodalVisionResultCarriesContentParts(t *testing.T) {
	reg := tools.NewRegistry()
	reg.MustRegister(&tools.MockTool{
		NameStr: "vision_analyze",
		ExecuteFn: func(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
			return json.RawMessage(`{
				"_multimodal": true,
				"text_summary": "Image attached natively for the main model.",
				"content": [
					{"type":"text","text":"Image loaded into your context."},
					{"type":"image_url","image_url":{"url":"data:image/png;base64,AAA"}}
				]
			}`), nil
		},
	})
	k := newKernelWithRegistry(t, reg)

	res := k.executeToolCalls(context.Background(), []llm.ToolCall{{
		ID:        "call_vision",
		Name:      "vision_analyze",
		Arguments: json.RawMessage(`{"image_url":"/tmp/x.png","question":"describe"}`),
	}})
	if len(res) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(res))
	}
	got := res[0]
	if got.Content != "Image attached natively for the main model." {
		t.Fatalf("Content = %q, want text summary fallback", got.Content)
	}
	if len(got.ContentParts) != 2 {
		t.Fatalf("ContentParts len = %d, want 2: %+v", len(got.ContentParts), got.ContentParts)
	}
	if got.ContentParts[0].Type != "text" || got.ContentParts[0].Text == "" {
		t.Fatalf("text part = %+v", got.ContentParts[0])
	}
	if got.ContentParts[1].Type != "image_url" || got.ContentParts[1].ImageURL != "data:image/png;base64,AAA" {
		t.Fatalf("image part = %+v", got.ContentParts[1])
	}
}
