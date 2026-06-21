package llm

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestOpenRouterRuntimeResolutionHonorsProviderAndCustomBase(t *testing.T) {
	tests := []struct {
		name        string
		req         OpenRouterRuntimeRequest
		env         map[string]string
		wantRuntime OpenRouterRuntime
		wantMissing bool
	}{
		{
			name: "explicit openrouter uses default base and openrouter key",
			req:  OpenRouterRuntimeRequest{Provider: "openrouter"},
			env:  map[string]string{"OPENROUTER_API_KEY": "or-key", "OPENAI_API_KEY": "openai-key"},
			wantRuntime: OpenRouterRuntime{
				Provider: "openrouter",
				BaseURL:  OpenRouterDefaultBaseURL,
				APIKey:   "or-key",
				Source:   OpenRouterRuntimeSourceEnvDefault,
			},
		},
		{
			name: "env base url is honored",
			req:  OpenRouterRuntimeRequest{Provider: "openrouter"},
			env:  map[string]string{"OPENROUTER_API_KEY": "or-key", "OPENROUTER_BASE_URL": "https://openrouter.ai/api/v1/"},
			wantRuntime: OpenRouterRuntime{
				Provider: "openrouter",
				BaseURL:  OpenRouterDefaultBaseURL,
				APIKey:   "or-key",
				Source:   OpenRouterRuntimeSourceEnvDefault,
			},
		},
		{
			name: "custom provider with openrouter base keeps custom label",
			req:  OpenRouterRuntimeRequest{Provider: "custom", BaseURL: "https://openrouter.ai/api/v1", APIKey: "explicit-key"},
			env:  map[string]string{"OPENROUTER_API_KEY": "or-key"},
			wantRuntime: OpenRouterRuntime{
				Provider: "custom",
				BaseURL:  OpenRouterDefaultBaseURL,
				APIKey:   "explicit-key",
				Source:   OpenRouterRuntimeSourceExplicit,
			},
		},
		{
			name:        "missing openrouter key returns setup evidence",
			req:         OpenRouterRuntimeRequest{Provider: "openrouter"},
			env:         map[string]string{},
			wantMissing: true,
			wantRuntime: OpenRouterRuntime{
				Provider: "openrouter",
				BaseURL:  OpenRouterDefaultBaseURL,
				Source:   OpenRouterRuntimeSourceEnvDefault,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.req
			req.LookupEnv = mapLookupEnv(tt.env)
			got := ResolveOpenRouterRuntime(req)
			if got.Provider != tt.wantRuntime.Provider {
				t.Fatalf("Provider = %q, want %q", got.Provider, tt.wantRuntime.Provider)
			}
			if got.BaseURL != tt.wantRuntime.BaseURL {
				t.Fatalf("BaseURL = %q, want %q", got.BaseURL, tt.wantRuntime.BaseURL)
			}
			if got.APIKey != tt.wantRuntime.APIKey {
				t.Fatalf("APIKey = %q, want %q", got.APIKey, tt.wantRuntime.APIKey)
			}
			if got.Source != tt.wantRuntime.Source {
				t.Fatalf("Source = %q, want %q", got.Source, tt.wantRuntime.Source)
			}
			if got.MissingAPIKey != tt.wantMissing {
				t.Fatalf("MissingAPIKey = %v, want %v", got.MissingAPIKey, tt.wantMissing)
			}
			if got.IsOpenRouterRoute != true {
				t.Fatalf("IsOpenRouterRoute = false, want true")
			}
		})
	}
}

func TestOpenRouterAttributionHeaders(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "https://example.test/v1/chat/completions", nil)
	if err != nil {
		t.Fatal(err)
	}
	ApplyOpenRouterAttributionHeaders(req, "openrouter", "https://example.test/v1")
	// X-Title is the canonical OpenRouter dashboard header (X-OpenRouter-Title was not recognized).
	// Mirrors Hermes fix(openrouter): use canonical X-Title attribution header (6430d6756).
	for _, header := range []string{"HTTP-Referer", "X-Title", "X-OpenRouter-Categories"} {
		if got := req.Header.Get(header); got == "" {
			t.Fatalf("%s header is empty, want OpenRouter attribution", header)
		}
	}
	if got := req.Header.Get("X-Title"); strings.Contains(strings.ToLower(got), "hermes") {
		t.Fatalf("X-Title = %q leaks Hermes label; Gormes should own OpenRouter attribution", got)
	}
	if got := req.Header.Get("X-OpenRouter-Title"); got != "" {
		t.Fatalf("X-OpenRouter-Title = %q, want empty (replaced by canonical X-Title)", got)
	}

	customReq, err := http.NewRequest(http.MethodPost, "https://openrouter.ai/api/v1/chat/completions", nil)
	if err != nil {
		t.Fatal(err)
	}
	ApplyOpenRouterAttributionHeaders(customReq, "custom", "https://openrouter.ai/api/v1")
	if customReq.Header.Get("HTTP-Referer") == "" {
		t.Fatal("custom provider with OpenRouter base URL should still get OpenRouter attribution headers")
	}

	plainReq, err := http.NewRequest(http.MethodPost, "https://example.test/v1/chat/completions", nil)
	if err != nil {
		t.Fatal(err)
	}
	ApplyOpenRouterAttributionHeaders(plainReq, "custom", "https://example.test/v1")
	for _, header := range []string{"HTTP-Referer", "X-Title", "X-OpenRouter-Title", "X-OpenRouter-Categories"} {
		if got := plainReq.Header.Get(header); got != "" {
			t.Fatalf("%s = %q, want no OpenRouter attribution for non-OpenRouter route", header, got)
		}
	}
}

func TestOpenRouterGrokPromptCacheAffinityHeader(t *testing.T) {
	tests := []struct {
		name      string
		provider  string
		model     string
		sessionID string
		want      string
	}{
		{
			name:      "x-ai grok model gets session affinity",
			provider:  "openrouter",
			model:     "x-ai/grok-4",
			sessionID: "sess-abc123",
			want:      "sess-abc123",
		},
		{
			name:      "xai grok model gets session affinity",
			provider:  "openrouter",
			model:     "xai/grok-3",
			sessionID: "sess-xyz",
			want:      "sess-xyz",
		},
		{
			name:      "non grok model omits header",
			provider:  "openrouter",
			model:     "anthropic/claude-sonnet-4.6",
			sessionID: "sess-abc123",
		},
		{
			name:     "grok without session omits header",
			provider: "openrouter",
			model:    "x-ai/grok-4",
		},
		{
			name:      "non openrouter route omits header",
			provider:  "custom",
			model:     "x-ai/grok-4",
			sessionID: "sess-abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var captured http.Header
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				captured = r.Header.Clone()
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)
				bw := bufio.NewWriter(w)
				_, _ = fmt.Fprint(bw, "data: {\"choices\":[{\"finish_reason\":\"stop\"}]}\n\n")
				_, _ = fmt.Fprint(bw, "data: [DONE]\n\n")
				_ = bw.Flush()
			}))
			defer srv.Close()

			baseURL := srv.URL
			client := NewHTTPClientWithProvider(baseURL, "test-key", tt.provider)
			stream, err := client.OpenStream(context.Background(), ChatRequest{
				Model:     tt.model,
				SessionID: tt.sessionID,
				Messages:  []Message{{Role: "user", Content: "hi"}},
			})
			if err != nil {
				t.Fatalf("OpenStream() error = %v", err)
			}
			defer stream.Close()

			if got := captured.Get("x-grok-conv-id"); got != tt.want {
				t.Fatalf("x-grok-conv-id = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOpenRouterModelMetadataAndPricing(t *testing.T) {
	body, err := os.ReadFile("testdata/openrouter/models.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	entries, err := ParseOpenRouterModelRegistry(body, "openrouter-models-api-test")
	if err != nil {
		t.Fatalf("ParseOpenRouterModelRegistry: %v", err)
	}
	registry := NewStaticModelRegistry(ModelRegistrySnapshot{Source: ModelRegistrySourceTestdata}, entries)
	got := registry.Lookup(ModelRegistryQuery{Provider: "openrouter", Model: "anthropic/claude-sonnet-4:free"})
	if !got.Found {
		t.Fatal("OpenRouter metadata lookup failed for suffix-bearing model")
	}
	if got.Model != "anthropic/claude-sonnet-4:free" {
		t.Fatalf("Model = %q, want suffix preserved", got.Model)
	}
	if got.RawContextWindow != 200000 || got.MaxOutputTokens != 32000 {
		t.Fatalf("limits = context %d output %d, want 200000/32000", got.RawContextWindow, got.MaxOutputTokens)
	}
	if got.Pricing.Source != ModelPricingSourceProviderModelsAPI {
		t.Fatalf("Pricing.Source = %q, want %q", got.Pricing.Source, ModelPricingSourceProviderModelsAPI)
	}
	if got.Pricing.InputUSDPerMillion != 3 || got.Pricing.OutputUSDPerMillion != 15 {
		t.Fatalf("pricing = input %v output %v, want 3/15 USD per million", got.Pricing.InputUSDPerMillion, got.Pricing.OutputUSDPerMillion)
	}
	if got.Capabilities.Tools != ModelCapabilitySupported || got.Capabilities.StructuredOutput != ModelCapabilitySupported {
		t.Fatalf("capabilities = %+v, want tools and structured output supported", got.Capabilities)
	}
}

func TestOpenRouterErrorClassification(t *testing.T) {
	rateLimit := newHTTPError(http.StatusTooManyRequests, `{"error":{"message":"Provider returned error","metadata":{"raw":"{\"error\":{\"message\":\"too many requests\",\"code\":\"rate_limit\"}}"}}}`, http.Header{"Retry-After": []string{"12"}})
	if rateLimit.RetryAfter != 12*time.Second {
		t.Fatalf("RetryAfter = %v, want 12s", rateLimit.RetryAfter)
	}
	classified := ClassifyProviderError(rateLimit)
	if classified.Kind != ProviderErrorRateLimit || !classified.Retryable || !classified.ShouldRotateCredential || !classified.ShouldFallback {
		t.Fatalf("rate-limit classification = %+v, want retryable rotating fallback", classified)
	}

	auth := ClassifyProviderError(newHTTPError(http.StatusUnauthorized, `{"error":{"message":"No auth credentials found"}}`, nil))
	if auth.Kind != ProviderErrorAuth || auth.Retryable {
		t.Fatalf("auth classification = %+v, want fatal auth", auth)
	}

	retryable := ClassifyProviderError(newHTTPError(http.StatusServiceUnavailable, `<html><body>upstream unavailable</body></html>`, nil))
	if retryable.Kind != ProviderErrorRetryable || !retryable.Retryable || strings.Contains(retryable.Message, "<html") {
		t.Fatalf("5xx classification = %+v, want sanitized retryable provider error", retryable)
	}
}

func mapLookupEnv(values map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	}
}

func TestOpenRouterAdaptiveAnthropicVerbosity(t *testing.T) {
	const orURL = "https://openrouter.ai/api/v1"
	const orProv = "openrouter"

	effortPtr := func(e ReasoningEffort) *ReasoningEffort { return &e }

	tests := []struct {
		name     string
		model    string
		effort   *ReasoningEffort
		wantVerb string
	}{
		// Adaptive models (Claude 4.6+) — should get verbosity
		{"sonnet-4-6 medium", "anthropic/claude-sonnet-4-6", effortPtr(ReasoningEffortMedium), "medium"},
		{"sonnet-4-6 high", "anthropic/claude-sonnet-4-6", effortPtr(ReasoningEffortHigh), "high"},
		{"sonnet-4-6 xhigh clamped to high", "anthropic/claude-sonnet-4-6", effortPtr(ReasoningEffortXHigh), "high"},
		{"sonnet-4-6 low", "anthropic/claude-sonnet-4-6", effortPtr(ReasoningEffortLow), "low"},
		{"sonnet-4-6 minimal→low", "anthropic/claude-sonnet-4-6", effortPtr(ReasoningEffortMinimal), "low"},
		{"fable-5 model", "anthropic/claude-fable-5", effortPtr(ReasoningEffortHigh), "high"},
		{"bare claude prefix", "claude-sonnet-4-6-20260101", effortPtr(ReasoningEffortMedium), "medium"},
		// none effort → no verbosity (can't disable adaptive thinking; omit)
		{"sonnet-4-6 none→no verb", "anthropic/claude-sonnet-4-6", effortPtr(ReasoningEffortNone), ""},
		// nil effort → no verbosity
		{"sonnet-4-6 nil effort", "anthropic/claude-sonnet-4-6", nil, ""},
		// Optional-reasoning models — no verbosity; they use reasoning.effort
		{"claude-3.5-sonnet", "anthropic/claude-3-5-sonnet", effortPtr(ReasoningEffortHigh), ""},
		{"opus-4-5", "anthropic/claude-opus-4-5", effortPtr(ReasoningEffortHigh), ""},
		{"sonnet-4-5", "anthropic/claude-sonnet-4-5", effortPtr(ReasoningEffortHigh), ""},
		{"haiku-4-5", "anthropic/claude-haiku-4-5", effortPtr(ReasoningEffortHigh), ""},
		// Non-Anthropic models — no verbosity
		{"gpt-4o", "openai/gpt-4o", effortPtr(ReasoningEffortHigh), ""},
		{"grok", "x-ai/grok-2", effortPtr(ReasoningEffortHigh), ""},
		// Non-OpenRouter route — no verbosity
		{"non-openrouter", "anthropic/claude-sonnet-4-6", effortPtr(ReasoningEffortHigh), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prov, url := orProv, orURL
			if tt.name == "non-openrouter" {
				prov, url = "openai", "https://api.openai.com/v1"
			}
			got := openRouterAdaptiveAnthropicVerbosity(prov, url, tt.model, tt.effort)
			if got != tt.wantVerb {
				t.Errorf("openRouterAdaptiveAnthropicVerbosity(%q, %q, effort=%v) = %q, want %q",
					url, tt.model, tt.effort, got, tt.wantVerb)
			}
		})
	}
}
