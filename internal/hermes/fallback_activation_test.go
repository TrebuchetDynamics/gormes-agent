package hermes

import (
	"encoding/json"
	"testing"
)

func TestFallbackConfigNormalizesOrderedList(t *testing.T) {
	policy := NormalizeFallbackModelConfig([]any{
		map[string]any{"provider": " OpenAI-Codex ", "model": " gpt-5.5 "},
		map[string]any{"provider": "", "model": "missing-provider"},
		map[string]any{"provider": "zai"},
		"not-a-route",
		map[string]any{"provider": " kimi-coding ", "model": " kimi-k2.5 "},
	})

	if !policy.Enabled {
		t.Fatal("Enabled = false, want true for valid fallback routes")
	}
	if len(policy.Routes) != 2 {
		t.Fatalf("Routes len = %d, want 2 valid routes", len(policy.Routes))
	}
	if got, want := policy.Routes[0], (ModelRoute{Provider: "openai-codex", Model: "gpt-5.5"}); got != want {
		t.Fatalf("Routes[0] = %#v, want %#v", got, want)
	}
	if got, want := policy.Routes[1], (ModelRoute{Provider: "kimi-coding", Model: "kimi-k2.5"}); got != want {
		t.Fatalf("Routes[1] = %#v, want %#v", got, want)
	}
}

func TestFallbackConfigNormalizesSingleObject(t *testing.T) {
	policy := NormalizeFallbackModelConfig(map[string]any{
		"provider": " ZAI ",
		"model":    " glm-5 ",
	})

	if !policy.Enabled {
		t.Fatal("Enabled = false, want true for single fallback object")
	}
	if len(policy.Routes) != 1 {
		t.Fatalf("Routes len = %d, want 1", len(policy.Routes))
	}
	if got, want := policy.Routes[0], (ModelRoute{Provider: "zai", Model: "glm-5"}); got != want {
		t.Fatalf("Routes[0] = %#v, want %#v", got, want)
	}
}

func TestFallbackJSONDecodeErrorRetryable(t *testing.T) {
	var decoded any
	err := json.Unmarshal([]byte(`{"bad":`), &decoded)
	if err == nil {
		t.Fatal("json.Unmarshal error = nil, want syntax error fixture")
	}

	classification := ClassifyProviderError(err)
	if classification.Kind != ProviderErrorRetryable {
		t.Fatalf("Kind = %q, want %q", classification.Kind, ProviderErrorRetryable)
	}
	if classification.Class != ClassRetryable {
		t.Fatalf("Class = %q, want retryable", classification.Class)
	}
	if !classification.Retryable {
		t.Fatal("Retryable = false, want true")
	}
	if !classification.ShouldFallback {
		t.Fatal("ShouldFallback = false, want malformed provider JSON eligible for fallback")
	}
}
