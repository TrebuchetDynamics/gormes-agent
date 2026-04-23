package hermes

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

const inlinePNGDataURL = "data:image/png;base64,aGVsbG8="

func decodeJSONMap(t *testing.T, v any) map[string]any {
	t.Helper()

	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}

func TestBuildAnthropicRequest_MapsTextAndInlineImageParts(t *testing.T) {
	req, err := buildAnthropicRequest(ChatRequest{
		Model: "claude-sonnet-4",
		Messages: []Message{{
			Role:    "user",
			Content: "Describe this image.",
			Parts: []ContentPart{{
				Type:     ContentPartImage,
				ImageURL: inlinePNGDataURL,
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(req.Messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(req.Messages))
	}
	content := req.Messages[0].Content
	if len(content) != 2 {
		t.Fatalf("content len = %d, want 2", len(content))
	}
	if content[0].Type != "text" || content[0].Text != "Describe this image." {
		t.Fatalf("content[0] = %+v, want text block", content[0])
	}
	if content[1].Type != "image" {
		t.Fatalf("content[1].type = %q, want image", content[1].Type)
	}
	if content[1].Source == nil {
		t.Fatal("content[1].source = nil, want base64 image source")
	}
	if content[1].Source.Type != "base64" {
		t.Fatalf("content[1].source.type = %q, want base64", content[1].Source.Type)
	}
	if content[1].Source.MediaType != "image/png" {
		t.Fatalf("content[1].source.media_type = %q, want image/png", content[1].Source.MediaType)
	}
	if content[1].Source.Data != "aGVsbG8=" {
		t.Fatalf("content[1].source.data = %q, want aGVsbG8=", content[1].Source.Data)
	}
}

func TestBuildGeminiRequest_MapsTextAndInlineImageParts(t *testing.T) {
	req, err := buildGeminiRequest(ChatRequest{
		Model: "gemini-2.5-flash",
		Messages: []Message{{
			Role:    "user",
			Content: "Describe this image.",
			Parts: []ContentPart{{
				Type:     ContentPartImage,
				ImageURL: inlinePNGDataURL,
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(req.Contents) != 1 {
		t.Fatalf("contents len = %d, want 1", len(req.Contents))
	}
	parts := req.Contents[0].Parts
	if len(parts) != 2 {
		t.Fatalf("parts len = %d, want 2", len(parts))
	}
	if parts[0].Text != "Describe this image." {
		t.Fatalf("parts[0].text = %q, want Describe this image.", parts[0].Text)
	}
	if parts[1].InlineData == nil {
		t.Fatal("parts[1].inlineData = nil, want image payload")
	}
	if parts[1].InlineData.MimeType != "image/png" {
		t.Fatalf("parts[1].inlineData.mimeType = %q, want image/png", parts[1].InlineData.MimeType)
	}
	if parts[1].InlineData.Data != "aGVsbG8=" {
		t.Fatalf("parts[1].inlineData.data = %q, want aGVsbG8=", parts[1].InlineData.Data)
	}
}

func TestBuildCodexRequest_MapsTextAndInlineImageParts(t *testing.T) {
	req, err := buildCodexRequest(ChatRequest{
		Model: "gpt-5.3-codex",
		Messages: []Message{{
			Role:    "user",
			Content: "Describe this image.",
			Parts: []ContentPart{{
				Type:     ContentPartImage,
				ImageURL: inlinePNGDataURL,
				Detail:   "high",
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	body := decodeJSONMap(t, req)
	input, ok := body["input"].([]any)
	if !ok {
		t.Fatalf("input = %#v, want array", body["input"])
	}
	if len(input) != 1 {
		t.Fatalf("input len = %d, want 1", len(input))
	}

	content, ok := input[0].(map[string]any)["content"].([]any)
	if !ok {
		t.Fatalf("input[0].content = %#v, want multimodal content array", input[0].(map[string]any)["content"])
	}
	if len(content) != 2 {
		t.Fatalf("content len = %d, want 2", len(content))
	}
	if got := content[0].(map[string]any)["type"]; got != "input_text" {
		t.Fatalf("content[0].type = %#v, want input_text", got)
	}
	if got := content[0].(map[string]any)["text"]; got != "Describe this image." {
		t.Fatalf("content[0].text = %#v, want Describe this image.", got)
	}
	if got := content[1].(map[string]any)["type"]; got != "input_image" {
		t.Fatalf("content[1].type = %#v, want input_image", got)
	}
	if got := content[1].(map[string]any)["image_url"]; got != inlinePNGDataURL {
		t.Fatalf("content[1].image_url = %#v, want %q", got, inlinePNGDataURL)
	}
	if got := content[1].(map[string]any)["detail"]; got != "high" {
		t.Fatalf("content[1].detail = %#v, want high", got)
	}
}

func TestBuildOpenRouterRequest_MapsTextAndInlineImageParts(t *testing.T) {
	req, err := buildOpenRouterRequest(ChatRequest{
		Model: "openai/gpt-4.1-mini",
		Messages: []Message{{
			Role:    "user",
			Content: "Describe this image.",
			Parts: []ContentPart{{
				Type:     ContentPartImage,
				ImageURL: inlinePNGDataURL,
				Detail:   "high",
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	body := decodeJSONMap(t, req)
	messages, ok := body["messages"].([]any)
	if !ok {
		t.Fatalf("messages = %#v, want array", body["messages"])
	}
	if len(messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(messages))
	}
	content, ok := messages[0].(map[string]any)["content"].([]any)
	if !ok {
		t.Fatalf("messages[0].content = %#v, want multimodal content array", messages[0].(map[string]any)["content"])
	}
	if len(content) != 2 {
		t.Fatalf("content len = %d, want 2", len(content))
	}
	if got := content[0].(map[string]any)["type"]; got != "text" {
		t.Fatalf("content[0].type = %#v, want text", got)
	}
	if got := content[0].(map[string]any)["text"]; got != "Describe this image." {
		t.Fatalf("content[0].text = %#v, want Describe this image.", got)
	}
	imagePart := content[1].(map[string]any)
	if got := imagePart["type"]; got != "image_url" {
		t.Fatalf("content[1].type = %#v, want image_url", got)
	}
	imageURL, ok := imagePart["image_url"].(map[string]any)
	if !ok {
		t.Fatalf("content[1].image_url = %#v, want object", imagePart["image_url"])
	}
	if got := imageURL["url"]; got != inlinePNGDataURL {
		t.Fatalf("content[1].image_url.url = %#v, want %q", got, inlinePNGDataURL)
	}
	if got := imageURL["detail"]; got != "high" {
		t.Fatalf("content[1].image_url.detail = %#v, want high", got)
	}
}

func TestBuildBedrockRequest_MapsTextAndInlineImageParts(t *testing.T) {
	req, err := buildBedrockRequest(ChatRequest{
		Model: "us.anthropic.claude-sonnet-4-6",
		Messages: []Message{{
			Role:    "user",
			Content: "Describe this image.",
			Parts: []ContentPart{{
				Type:     ContentPartImage,
				ImageURL: inlinePNGDataURL,
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(req.Messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(req.Messages))
	}
	content := req.Messages[0].Content
	if len(content) != 2 {
		t.Fatalf("content len = %d, want 2", len(content))
	}
	textBlock, ok := content[0].(*types.ContentBlockMemberText)
	if !ok || textBlock.Value != "Describe this image." {
		t.Fatalf("content[0] = %#v, want text block", content[0])
	}
	imageBlock, ok := content[1].(*types.ContentBlockMemberImage)
	if !ok {
		t.Fatalf("content[1] = %#v, want image block", content[1])
	}
	if imageBlock.Value.Format != types.ImageFormat("png") {
		t.Fatalf("image format = %q, want png", imageBlock.Value.Format)
	}
	imageSource, ok := imageBlock.Value.Source.(*types.ImageSourceMemberBytes)
	if !ok {
		t.Fatalf("image source = %#v, want byte source", imageBlock.Value.Source)
	}
	if string(imageSource.Value) != "hello" {
		t.Fatalf("image bytes = %q, want hello", string(imageSource.Value))
	}
}

func TestHTTPClientOpenStream_MapsTextAndInlineImageParts(t *testing.T) {
	reqSeen := make(chan map[string]any, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		reqSeen <- body

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		bw := bufio.NewWriter(w)
		fmt.Fprint(bw, sseHappy)
		bw.Flush()
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "")
	s, err := c.OpenStream(context.Background(), ChatRequest{
		Model: "hermes-agent",
		Messages: []Message{{
			Role:    "user",
			Content: "Describe this image.",
			Parts: []ContentPart{{
				Type:     ContentPartImage,
				ImageURL: inlinePNGDataURL,
				Detail:   "high",
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	for {
		ev, err := s.Recv(context.Background())
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if ev.Kind == EventDone {
			break
		}
	}

	body := <-reqSeen
	messages, ok := body["messages"].([]any)
	if !ok {
		t.Fatalf("messages = %#v, want array", body["messages"])
	}
	if len(messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(messages))
	}
	content, ok := messages[0].(map[string]any)["content"].([]any)
	if !ok {
		t.Fatalf("messages[0].content = %#v, want multimodal content array", messages[0].(map[string]any)["content"])
	}
	if len(content) != 2 {
		t.Fatalf("content len = %d, want 2", len(content))
	}
	if got := content[0].(map[string]any)["type"]; got != "text" {
		t.Fatalf("content[0].type = %#v, want text", got)
	}
	imagePart := content[1].(map[string]any)
	if got := imagePart["type"]; got != "image_url" {
		t.Fatalf("content[1].type = %#v, want image_url", got)
	}
	imageURL, ok := imagePart["image_url"].(map[string]any)
	if !ok {
		t.Fatalf("content[1].image_url = %#v, want object", imagePart["image_url"])
	}
	if got := imageURL["url"]; got != inlinePNGDataURL {
		t.Fatalf("content[1].image_url.url = %#v, want %q", got, inlinePNGDataURL)
	}
	if got := imageURL["detail"]; got != "high" {
		t.Fatalf("content[1].image_url.detail = %#v, want high", got)
	}
}
