package llm

import "testing"

func TestNormalizeModelForProviderMatchesHermesExamples(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		model    string
		provider string
		want     string
	}{
		{name: "openrouter prepends detected vendor", model: "claude-sonnet-4.6", provider: "openrouter", want: "anthropic/claude-sonnet-4.6"},
		{name: "aggregator preserves existing vendor", model: "anthropic/claude-sonnet-4.6", provider: "openrouter", want: "anthropic/claude-sonnet-4.6"},
		{name: "anthropic strips matching prefix and hyphenates dots", model: "anthropic/claude-sonnet-4.6", provider: "anthropic", want: "claude-sonnet-4-6"},
		{name: "opencode zen hyphenates claude", model: "claude-sonnet-4.6", provider: "opencode-zen", want: "claude-sonnet-4-6"},
		{name: "opencode go strips arbitrary vendor and preserves dot version", model: "minimax/minimax-m2.7", provider: "opencode-go", want: "minimax-m2.7"},
		{name: "copilot strips anthropic prefix and keeps dots", model: "anthropic/claude-sonnet-4.6", provider: "copilot", want: "claude-sonnet-4.6"},
		{name: "copilot repairs claude dash notation", model: "claude-sonnet-4-6", provider: "copilot-acp", want: "claude-sonnet-4.6"},
		{name: "openai codex strips openai prefix", model: "openai/gpt-5.4", provider: "openai-codex", want: "gpt-5.4"},
		{name: "deepseek v-series model passes through", model: "deepseek-v3", provider: "deepseek", want: "deepseek-v3"},
		{name: "deepseek v4 model passes through", model: "deepseek/deepseek-v4-pro", provider: "deepseek", want: "deepseek-v4-pro"},
		{name: "matching native prefix strips zai", model: "zai/glm-5.1", provider: "zai", want: "glm-5.1"},
		{name: "provider aliases match for kimi", model: "moonshot/kimi-k2.5", provider: "kimi-coding", want: "kimi-k2.5"},
		{name: "huggingface preserves native casing", model: "Qwen/Qwen3.5-397B-A17B", provider: "huggingface", want: "Qwen/Qwen3.5-397B-A17B"},
		{name: "xiaomi lowercases model", model: "MiMo-V2.5-Pro", provider: "xiaomi", want: "mimo-v2.5-pro"},
		{name: "custom preserves slash-bearing model", model: "modal/zai-org/GLM-5-FP8", provider: "custom", want: "modal/zai-org/GLM-5-FP8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeModelForProvider(tt.model, tt.provider); got != tt.want {
				t.Fatalf("NormalizeModelForProvider(%q, %q) = %q, want %q", tt.model, tt.provider, got, tt.want)
			}
		})
	}
}

func TestDetectModelVendorMatchesHermesPrefixes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		model string
		want  string
	}{
		{model: "claude-sonnet-4.6", want: "anthropic"},
		{model: "gpt-5.4-mini", want: "openai"},
		{model: "minimax-m2.7", want: "minimax"},
		{model: "glm-4.5", want: "z-ai"},
		{model: "kimi-k2.5", want: "moonshotai"},
		{model: "anthropic/claude-sonnet-4.6", want: "anthropic"},
		{model: "my-custom-model", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			t.Parallel()
			if got := DetectModelVendor(tt.model); got != tt.want {
				t.Fatalf("DetectModelVendor(%q) = %q, want %q", tt.model, got, tt.want)
			}
		})
	}
}
