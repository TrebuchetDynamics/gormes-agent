package llm

import "testing"

func TestOpenRouterParetoPluginSerializesForParetoModel(t *testing.T) {
	client := NewHTTPClientWithProvider("https://openrouter.ai/api/v1", "test-key", "openrouter").(*httpClient)
	body, _, err := client.buildOpenAICompatibleChatRequestBody(ChatRequest{
		Model:    "openrouter/pareto-code",
		Messages: []Message{{Role: "user", Content: "hi"}},
		RequestOverrides: RequestOverrides{
			OpenRouterMinCodingScore: "0.65",
		},
	})
	if err != nil {
		t.Fatalf("buildOpenAICompatibleChatRequestBody() error = %v", err)
	}
	plugin, ok := openRouterParetoPluginFromBody(t, body)
	if !ok {
		t.Fatalf("Pareto plugin missing from body: %s", body)
	}
	if got := plugin["id"]; got != "pareto-router" {
		t.Fatalf("plugin id = %#v, want pareto-router", got)
	}
	if got := plugin["min_coding_score"]; got != 0.65 {
		t.Fatalf("min_coding_score = %#v, want 0.65", got)
	}
}

func TestOpenRouterParetoPluginOmittedOutsideScope(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		baseURL  string
		model    string
		score    string
	}{
		{name: "non pareto model", provider: "openrouter", baseURL: "https://openrouter.ai/api/v1", model: "anthropic/claude-sonnet-4.6", score: "0.65"},
		{name: "non openrouter provider", provider: "custom", baseURL: "https://example.test/v1", model: "openrouter/pareto-code", score: "0.65"},
		{name: "unset", provider: "openrouter", baseURL: "https://openrouter.ai/api/v1", model: "openrouter/pareto-code", score: ""},
		{name: "invalid", provider: "openrouter", baseURL: "https://openrouter.ai/api/v1", model: "openrouter/pareto-code", score: "not-a-number"},
		{name: "nan", provider: "openrouter", baseURL: "https://openrouter.ai/api/v1", model: "openrouter/pareto-code", score: "NaN"},
		{name: "inf", provider: "openrouter", baseURL: "https://openrouter.ai/api/v1", model: "openrouter/pareto-code", score: "+Inf"},
		{name: "negative", provider: "openrouter", baseURL: "https://openrouter.ai/api/v1", model: "openrouter/pareto-code", score: "-0.1"},
		{name: "greater than one", provider: "openrouter", baseURL: "https://openrouter.ai/api/v1", model: "openrouter/pareto-code", score: "1.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewHTTPClientWithProvider(tt.baseURL, "test-key", tt.provider).(*httpClient)
			body, _, err := client.buildOpenAICompatibleChatRequestBody(ChatRequest{
				Model:    tt.model,
				Messages: []Message{{Role: "user", Content: "hi"}},
				RequestOverrides: RequestOverrides{
					OpenRouterMinCodingScore: tt.score,
				},
			})
			if err != nil {
				t.Fatalf("buildOpenAICompatibleChatRequestBody() error = %v", err)
			}
			if plugin, ok := openRouterParetoPluginFromBody(t, body); ok {
				t.Fatalf("unexpected Pareto plugin %#v in body: %s", plugin, body)
			}
		})
	}
}

func openRouterParetoPluginFromBody(t *testing.T, raw []byte) (map[string]any, bool) {
	t.Helper()
	body := decodeJSONMap(t, raw)
	extra, ok := body["extra_body"].(map[string]any)
	if !ok {
		return nil, false
	}
	plugins, ok := extra["plugins"].([]any)
	if !ok || len(plugins) == 0 {
		return nil, false
	}
	plugin, ok := plugins[0].(map[string]any)
	return plugin, ok
}
