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

func TestFallbackConfigPreservesCredentialAliases(t *testing.T) {
	policy := NormalizeFallbackModelConfig([]any{
		map[string]any{
			"provider":    " custom ",
			"model":       " gemini-flash ",
			"base_url":    " https://generativelanguage.googleapis.com/v1beta/openai ",
			"api_key_env": " MY_GOOGLE_KEY ",
		},
		map[string]any{
			"provider":    "openrouter",
			"model":       "some/model",
			"key_env":     " OPENROUTER_API_KEY ",
			"api_key_env": "SHOULD_NOT_WIN",
		},
		map[string]any{
			"provider": "zai",
			"model":    "glm-5",
			"api_key":  " inline-secret ",
			"api_mode": "responses",
		},
	})

	if len(policy.Routes) != 3 {
		t.Fatalf("Routes len = %d, want 3: %#v", len(policy.Routes), policy.Routes)
	}
	if got := policy.Routes[0]; got.Provider != "custom" ||
		got.Model != "gemini-flash" ||
		got.BaseURL != "https://generativelanguage.googleapis.com/v1beta/openai" ||
		got.APIKeyEnv != "MY_GOOGLE_KEY" ||
		got.KeyEnv != "" ||
		got.ExplicitAPIKey != "" {
		t.Fatalf("api_key_env route = %#v, want base_url plus api_key_env only", got)
	}
	if got := policy.Routes[1]; got.KeyEnv != "OPENROUTER_API_KEY" || got.APIKeyEnv != "SHOULD_NOT_WIN" {
		t.Fatalf("key_env route = %#v, want both aliases preserved", got)
	}
	if got := policy.Routes[2]; got.ExplicitAPIKey != "inline-secret" || got.APIMode != "responses" {
		t.Fatalf("inline api key route = %#v, want explicit key and api mode preserved", got)
	}
}

func TestFallbackRouteResolveCredentialAliases(t *testing.T) {
	lookup := func(name string) string {
		switch name {
		case "KEY_ENV":
			return "key-env-secret"
		case "API_KEY_ENV":
			return "api-key-env-secret"
		default:
			return ""
		}
	}

	t.Run("explicit key wins", func(t *testing.T) {
		route := ModelRoute{ExplicitAPIKey: "inline", KeyEnv: "KEY_ENV", APIKeyEnv: "API_KEY_ENV"}
		if got := route.ResolveFallbackCredential(lookup).ExplicitAPIKey; got != "inline" {
			t.Fatalf("ExplicitAPIKey = %q, want inline", got)
		}
	})
	t.Run("key env wins over api key env", func(t *testing.T) {
		route := ModelRoute{KeyEnv: "KEY_ENV", APIKeyEnv: "API_KEY_ENV"}
		if got := route.ResolveFallbackCredential(lookup).ExplicitAPIKey; got != "key-env-secret" {
			t.Fatalf("ExplicitAPIKey = %q, want key_env secret", got)
		}
	})
	t.Run("api key env fallback", func(t *testing.T) {
		route := ModelRoute{APIKeyEnv: "API_KEY_ENV"}
		if got := route.ResolveFallbackCredential(lookup).ExplicitAPIKey; got != "api-key-env-secret" {
			t.Fatalf("ExplicitAPIKey = %q, want api_key_env secret", got)
		}
	})
	t.Run("unset env stays absent", func(t *testing.T) {
		route := ModelRoute{APIKeyEnv: "ABSENT"}
		if got := route.ResolveFallbackCredential(lookup).ExplicitAPIKey; got != "" {
			t.Fatalf("ExplicitAPIKey = %q, want empty", got)
		}
	})
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
