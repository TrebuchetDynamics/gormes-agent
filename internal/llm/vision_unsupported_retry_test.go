package llm

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVisionUnsupportedRetryDetectorMatchesHermesPhrases(t *testing.T) {
	matches := []string{
		"only 'text' content type is supported",
		"only text content type is supported",
		"image_url is not supported",
		"image content is not supported",
		"multimodal is not supported",
		"multimodal content is not supported",
		"multimodal input is not supported",
		"vision is not supported",
		"vision input is not supported",
		"does not support images",
		"does not support image input",
		"does not support multimodal",
		"does not support vision",
		"model does not support image",
		"unsupported content type: image_url",
		"unknown variant image_url",
	}
	for _, phrase := range matches {
		t.Run(phrase, func(t *testing.T) {
			err := newHTTPError(http.StatusBadRequest, fmt.Sprintf(`{"error":{"message":"%s"}}`, phrase), nil)
			if !isVisionUnsupportedProviderError(err) {
				t.Fatalf("isVisionUnsupportedProviderError(%q) = false, want true", phrase)
			}
		})
	}

	nonMatches := []struct {
		name   string
		status int
		body   string
	}{
		{name: "server_error", status: http.StatusBadGateway, body: `{"error":{"message":"model does not support image"}}`},
		{name: "timeout_text", status: http.StatusGatewayTimeout, body: `{"error":{"message":"image_url is not supported"}}`},
		{name: "unrelated_bad_request", status: http.StatusBadRequest, body: `{"error":{"message":"Invalid value: tool role is not allowed"}}`},
	}
	for _, tt := range nonMatches {
		t.Run(tt.name, func(t *testing.T) {
			if isVisionUnsupportedProviderError(newHTTPError(tt.status, tt.body, nil)) {
				t.Fatalf("isVisionUnsupportedProviderError(status=%d, body=%s) = true, want false", tt.status, tt.body)
			}
		})
	}
}

func TestVisionUnsupportedRetryStripsImagesWithoutMutatingOriginal(t *testing.T) {
	original := []Message{
		{Role: "system", Content: "Follow policy."},
		{
			Role: "user",
			ContentParts: []MessageContentPart{
				{Type: "text", Text: "describe this"},
				{Type: "image_url", ImageURL: "data:image/png;base64,AAA", Detail: "high"},
				{Type: "input_image", ImageURL: "data:image/png;base64,BBB"},
			},
		},
		{
			Role:       "tool",
			ToolCallID: "call_image",
			Name:       "browser_vision",
			ContentParts: []MessageContentPart{
				{Type: "image", ImageURL: "data:image/png;base64,CCC"},
			},
		},
	}

	plan := PlanVisionUnsupportedRetry(VisionUnsupportedRetryRequest{
		Err:      newHTTPError(http.StatusBadRequest, `{"error":{"message":"does not support vision"}}`, nil),
		Messages: original,
		Attempts: 0,
	})

	if !plan.Retry {
		t.Fatalf("Retry = false, want true (evidence=%q)", plan.EvidenceCode)
	}
	if !plan.ImagesRemoved {
		t.Fatal("ImagesRemoved = false, want true")
	}
	if hasMessageImagePart(plan.NewMessages) {
		t.Fatalf("retry messages still contain image parts: %+v", plan.NewMessages)
	}
	if len(plan.NewMessages) != len(original) {
		t.Fatalf("NewMessages len = %d, want %d", len(plan.NewMessages), len(original))
	}
	if got := plan.NewMessages[1].ContentParts; len(got) != 1 || got[0].Text != "describe this" {
		t.Fatalf("user retry content parts = %+v, want only original text part", got)
	}
	if !strings.Contains(plan.NewMessages[2].Content, "image content removed") {
		t.Fatalf("tool retry content = %q, want image removal placeholder", plan.NewMessages[2].Content)
	}
	if !hasMessageImagePart(original) {
		t.Fatal("original messages were mutated; image parts disappeared from caller-owned request")
	}
}

func TestVisionUnsupportedRetryOpenAICompatibleRetriesOnceAndRemembersSession(t *testing.T) {
	var captured [][]byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		captured = append(captured, append([]byte(nil), raw...))
		if len(captured) == 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprint(w, `{"error":{"message":"Only 'text' content type is supported."}}`)
			return
		}
		if jsonHasKey(t, raw, "image_url") {
			t.Fatalf("request %d still contains image_url after session marked text-only: %s", len(captured), raw)
		}
		writeVisionRetrySSE(w)
	}))
	defer srv.Close()

	client := NewHTTPClientWithProvider(srv.URL, "test-key", "openrouter")
	req := visionUnsupportedRetryRequest("session-image-1")
	stream, err := client.OpenStream(context.Background(), req)
	if err != nil {
		t.Fatalf("OpenStream() first turn error = %v", err)
	}
	_ = stream.Close()

	if len(captured) != 2 {
		t.Fatalf("request count after retry = %d, want exactly 2", len(captured))
	}
	if !jsonHasKey(t, captured[0], "image_url") {
		t.Fatalf("first request lacked image_url; test is not exercising multimodal retry: %s", captured[0])
	}
	if jsonHasKey(t, captured[1], "image_url") {
		t.Fatalf("retry request still contains image_url: %s", captured[1])
	}
	if !hasMessageImagePart(req.Messages) {
		t.Fatal("OpenStream mutated caller-owned ChatRequest messages")
	}

	stream, err = client.OpenStream(context.Background(), visionUnsupportedRetryRequest("session-image-1"))
	if err != nil {
		t.Fatalf("OpenStream() remembered session error = %v", err)
	}
	_ = stream.Close()
	if len(captured) != 3 {
		t.Fatalf("request count after remembered session = %d, want 3", len(captured))
	}
	if jsonHasKey(t, captured[2], "image_url") {
		t.Fatalf("remembered text-only session emitted image_url before provider rejection: %s", captured[2])
	}
}

func TestVisionUnsupportedRetryOpenAICompatibleDoesNotRetry5xxOrUnrelated4xx(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{name: "5xx vision phrase", status: http.StatusBadGateway, body: `{"error":{"message":"image_url is not supported"}}`},
		{name: "unrelated 4xx", status: http.StatusBadRequest, body: `{"error":{"message":"Invalid role ordering"}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var requests int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requests++
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = fmt.Fprint(w, tc.body)
			}))
			defer srv.Close()

			client := NewHTTPClient(srv.URL, "")
			if _, err := client.OpenStream(context.Background(), visionUnsupportedRetryRequest("session-"+tc.name)); err == nil {
				t.Fatal("OpenStream() error = nil, want provider error")
			}
			if requests != 1 {
				t.Fatalf("request count = %d, want 1", requests)
			}
		})
	}
}

func TestVisionUnsupportedRetryCodexResponsesRetriesWithTextOnlyRequest(t *testing.T) {
	var captured [][]byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		captured = append(captured, append([]byte(nil), raw...))
		if len(captured) == 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprint(w, `{"error":{"message":"unsupported content type: image_url"}}`)
			return
		}
		if jsonHasKey(t, raw, "image_url") {
			t.Fatalf("Codex retry request still contains image_url: %s", raw)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status":"completed","output":[{"type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"ok"}]}],"usage":{"input_tokens":3,"output_tokens":1,"total_tokens":4}}`)
	}))
	defer srv.Close()

	client := NewHTTPClientWithProvider(srv.URL, "test-key", "openai-codex")
	stream, err := client.OpenStream(context.Background(), visionUnsupportedRetryRequest("codex-session-image-1"))
	if err != nil {
		t.Fatalf("OpenStream() error = %v", err)
	}
	_ = stream.Close()
	if len(captured) != 2 {
		t.Fatalf("request count = %d, want 2", len(captured))
	}
	if !jsonHasKey(t, captured[0], "image_url") {
		t.Fatalf("first Codex request lacked image_url: %s", captured[0])
	}
	if jsonHasKey(t, captured[1], "image_url") {
		t.Fatalf("retry Codex request still contains image_url: %s", captured[1])
	}
}

func visionUnsupportedRetryRequest(sessionID string) ChatRequest {
	return ChatRequest{
		Model:     "fixture-model",
		SessionID: sessionID,
		Stream:    true,
		Messages: []Message{
			{Role: "system", Content: "Follow policy."},
			{
				Role: "user",
				ContentParts: []MessageContentPart{
					{Type: "text", Text: "describe this"},
					{Type: "image_url", ImageURL: "data:image/png;base64,AAA", Detail: "high"},
				},
			},
		},
	}
}

func writeVisionRetrySSE(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	bw := bufio.NewWriter(w)
	_, _ = fmt.Fprint(bw, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
	_, _ = fmt.Fprint(bw, "data: {\"choices\":[{\"finish_reason\":\"stop\"}]}\n\n")
	_, _ = fmt.Fprint(bw, "data: [DONE]\n\n")
	_ = bw.Flush()
}

func hasMessageImagePart(messages []Message) bool {
	for _, msg := range messages {
		for _, part := range msg.ContentParts {
			switch strings.ToLower(strings.TrimSpace(part.Type)) {
			case "image_url", "input_image", "image":
				if strings.TrimSpace(part.ImageURL) != "" {
					return true
				}
			}
		}
	}
	return false
}
