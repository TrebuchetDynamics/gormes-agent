package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResolveFastModeRequestOverrides(t *testing.T) {
	tests := []struct {
		name   string
		model  string
		want   RequestOverrides
		wantOK bool
	}{
		{name: "openai_gpt_prefix", model: "gpt-5.4", want: RequestOverrides{ServiceTier: "priority"}, wantOK: true},
		{name: "openai_vendor_prefix", model: "openai/gpt-4.1", want: RequestOverrides{ServiceTier: "priority"}, wantOK: true},
		{name: "openai_o_prefix", model: "o3", want: RequestOverrides{ServiceTier: "priority"}, wantOK: true},
		{name: "codex_excluded", model: "gpt-5.3-codex", wantOK: false},
		{name: "claude_opus_46_dash", model: "claude-opus-4-6", want: RequestOverrides{Speed: "fast"}, wantOK: true},
		{name: "claude_opus_46_dot_vendor", model: "anthropic/claude-opus-4.6", want: RequestOverrides{Speed: "fast"}, wantOK: true},
		{name: "claude_sonnet_unsupported", model: "claude-sonnet-4-6", wantOK: false},
		{name: "claude_opus_47_unsupported", model: "claude-opus-4-7", wantOK: false},
		{name: "gemini_unsupported", model: "gemini-3-pro-preview", wantOK: false},
		{name: "empty_unsupported", model: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ResolveFastModeRequestOverrides(tt.model)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v; overrides = %+v", ok, tt.wantOK, got)
			}
			if got != tt.want {
				t.Fatalf("overrides = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestOpenAICompatibleChatRequestSerializesServiceTier(t *testing.T) {
	client := NewHTTPClientWithProvider("https://example.test", "test-key", "openai").(*httpClient)
	withOverride, _, err := client.buildOpenAICompatibleChatRequestBody(ChatRequest{
		Model:            "gpt-5.4",
		Messages:         []Message{{Role: "user", Content: "hi"}},
		RequestOverrides: RequestOverrides{ServiceTier: "priority"},
	})
	if err != nil {
		t.Fatalf("buildOpenAICompatibleChatRequestBody(with override) error = %v", err)
	}
	body := decodeJSONMap(t, withOverride)
	if got := body["service_tier"]; got != "priority" {
		t.Fatalf("service_tier = %#v, want priority in body: %s", got, withOverride)
	}

	withoutOverride, _, err := client.buildOpenAICompatibleChatRequestBody(ChatRequest{
		Model:    "gpt-5.4",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("buildOpenAICompatibleChatRequestBody(without override) error = %v", err)
	}
	body = decodeJSONMap(t, withoutOverride)
	if _, ok := body["service_tier"]; ok {
		t.Fatalf("service_tier present without explicit override: %s", withoutOverride)
	}
}

func TestCodexResponsesPayloadSerializesServiceTier(t *testing.T) {
	payload, err := buildCodexResponsesPayload(ChatRequest{
		Model:            "gpt-5-codex",
		Messages:         []Message{{Role: "user", Content: "hi"}},
		RequestOverrides: RequestOverrides{ServiceTier: "priority"},
	})
	if err != nil {
		t.Fatalf("buildCodexResponsesPayload() error = %v", err)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	body := decodeJSONMap(t, raw)
	if got := body["service_tier"]; got != "priority" {
		t.Fatalf("service_tier = %#v, want priority in payload: %s", got, raw)
	}

	if got, ok := ResolveFastModeRequestOverrides("gpt-5-codex"); ok || got != (RequestOverrides{}) {
		t.Fatalf("fast-mode resolver emitted overrides for codex model: ok=%v overrides=%+v", ok, got)
	}
}

func TestAnthropicFastModeSerializesOnlyEligibleSpeed(t *testing.T) {
	var capturedHeader http.Header
	var capturedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeader = r.Header.Clone()
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
	}))
	defer srv.Close()

	stream, err := NewAnthropicClient(srv.URL, "sk-ant-api-test").OpenStream(context.Background(), ChatRequest{
		Model:            "claude-opus-4-6",
		Messages:         []Message{{Role: "user", Content: "hi"}},
		RequestOverrides: RequestOverrides{Speed: "fast"},
	})
	if err != nil {
		t.Fatalf("OpenStream() error = %v", err)
	}
	if _, err := stream.Recv(context.Background()); err != io.EOF {
		t.Fatalf("Recv() err = %v, want EOF", err)
	}
	defer stream.Close()

	if got := capturedBody["speed"]; got != "fast" {
		t.Fatalf("speed = %#v, want fast in Anthropic body: %#v", got, capturedBody)
	}
	if got := capturedHeader.Get("anthropic-beta"); !strings.Contains(got, "fast-mode-2026-02-01") {
		t.Fatalf("anthropic-beta = %q, want fast-mode beta", got)
	}

	for _, model := range []string{"claude-opus-4-7", "claude-sonnet-4-6", "anthropic/claude-haiku-4-5"} {
		t.Run(model, func(t *testing.T) {
			payload, err := buildAnthropicRequest(ChatRequest{
				Model:            model,
				Messages:         []Message{{Role: "user", Content: "hi"}},
				RequestOverrides: RequestOverrides{Speed: "fast"},
			})
			if err != nil {
				t.Fatalf("buildAnthropicRequest() error = %v", err)
			}
			raw, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("marshal payload: %v", err)
			}
			if strings.Contains(string(raw), `"speed"`) {
				t.Fatalf("unsupported model payload contains speed: %s", raw)
			}
		})
	}
}
