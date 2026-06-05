package llm

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPromptCachePolicyMatchesHermesProviderMatrix(t *testing.T) {
	tests := []struct {
		name       string
		provider   string
		baseURL    string
		apiMode    string
		model      string
		wantCache  bool
		wantLayout PromptCacheLayout
	}{
		{
			name:       "native anthropic messages",
			provider:   "anthropic",
			baseURL:    "https://api.anthropic.com",
			apiMode:    "anthropic_messages",
			model:      "claude-sonnet-4-5",
			wantCache:  true,
			wantLayout: PromptCacheLayoutNativeAnthropic,
		},
		{
			name:       "openrouter claude chat completions",
			provider:   "openrouter",
			baseURL:    "https://openrouter.ai/api/v1",
			apiMode:    "chat_completions",
			model:      "anthropic/claude-sonnet-4-5",
			wantCache:  true,
			wantLayout: PromptCacheLayoutEnvelope,
		},
		{
			name:       "third party anthropic wire claude",
			provider:   "fixture-gateway",
			baseURL:    "https://anthropic-proxy.example.test",
			apiMode:    "anthropic_messages",
			model:      "claude-opus-4-1",
			wantCache:  true,
			wantLayout: PromptCacheLayoutNativeAnthropic,
		},
		{
			name:       "opencode qwen chat completions",
			provider:   "opencode-go",
			baseURL:    "https://dashscope.example.test/compatible-mode/v1",
			apiMode:    "chat_completions",
			model:      "qwen3.6-plus",
			wantCache:  true,
			wantLayout: PromptCacheLayoutEnvelope,
		},
		{
			name:       "generic openai wire model",
			provider:   "openai",
			baseURL:    "https://api.openai.com/v1",
			apiMode:    "chat_completions",
			model:      "gpt-5.1",
			wantCache:  false,
			wantLayout: PromptCacheLayoutUnsupported,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PromptCachePolicyFor(PromptCachePolicyInput{
				Provider: tt.provider,
				BaseURL:  tt.baseURL,
				APIMode:  tt.apiMode,
				Model:    tt.model,
			})
			if got.ShouldCache != tt.wantCache {
				t.Fatalf("ShouldCache = %v, want %v (%s)", got.ShouldCache, tt.wantCache, got.Reason)
			}
			if got.Layout != tt.wantLayout {
				t.Fatalf("Layout = %q, want %q", got.Layout, tt.wantLayout)
			}
			if strings.TrimSpace(got.Reason) == "" {
				t.Fatal("Reason is empty")
			}
		})
	}
}

func TestApplyPromptCacheControlSystemAndLastThree(t *testing.T) {
	messages := []Message{
		{Role: "system", Content: "stable system"},
		{Role: "user", Content: "one"},
		{Role: "assistant", Content: "two"},
		{Role: "user", Content: "three"},
		{Role: "assistant", Content: "four"},
	}

	got := ApplyPromptCacheControl(messages, PromptCachePolicy{ShouldCache: true, Layout: PromptCacheLayoutEnvelope, TTL: "1h"})
	if messages[0].CacheControl != nil || got[1].CacheControl != nil {
		t.Fatalf("ApplyPromptCacheControl mutated input or cached an old non-system message: input=%+v got=%+v", messages, got)
	}
	wantCached := map[int]bool{0: true, 2: true, 3: true, 4: true}
	for idx := range got {
		cached := got[idx].CacheControl != nil
		if cached != wantCached[idx] {
			t.Fatalf("message %d cached = %v, want %v; got=%+v", idx, cached, wantCached[idx], got)
		}
		if cached && got[idx].CacheControl.TTL != "1h" {
			t.Fatalf("message %d TTL = %q, want 1h", idx, got[idx].CacheControl.TTL)
		}
	}
}

func TestOpenAICompatiblePromptCachePolicySerializesOnlyAllowedEnvelopeMarkers(t *testing.T) {
	client := NewHTTPClientWithProvider("https://openrouter.ai/api/v1", "", "openrouter")
	req := ChatRequest{Model: "anthropic/claude-sonnet-4-5", Messages: []Message{
		{Role: "system", Content: "stable system"},
		{Role: "user", Content: "hello"},
	}}
	body, _, err := client.(*httpClient).buildOpenAICompatibleChatRequestBody(req)
	if err != nil {
		t.Fatalf("buildOpenAICompatibleChatRequestBody() error = %v", err)
	}
	if !strings.Contains(string(body), "cache_control") {
		t.Fatalf("body = %s, want cache_control for OpenRouter Claude", body)
	}

	var decoded struct {
		Messages []struct {
			Role    string `json:"role"`
			Content any    `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if len(decoded.Messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(decoded.Messages))
	}
	if _, ok := decoded.Messages[0].Content.([]any); !ok {
		t.Fatalf("system content = %#v, want cached content block envelope", decoded.Messages[0].Content)
	}

	generic := NewHTTPClientWithProvider("https://api.openai.com/v1", "", "openai")
	genericBody, _, err := generic.(*httpClient).buildOpenAICompatibleChatRequestBody(req)
	if err != nil {
		t.Fatalf("generic buildOpenAICompatibleChatRequestBody() error = %v", err)
	}
	if strings.Contains(string(genericBody), "cache_control") {
		t.Fatalf("generic body = %s, did not want unsupported cache_control", genericBody)
	}
	status := ProviderStatusOf(client)
	if !status.Capabilities.PromptCache.Available || !strings.Contains(status.Capabilities.PromptCache.Reason, "prompt_cache_supported") {
		t.Fatalf("PromptCache status = %+v, want supported OpenRouter Claude policy", status.Capabilities.PromptCache)
	}
}
