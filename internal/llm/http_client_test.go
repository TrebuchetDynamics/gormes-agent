package llm

import "testing"

func TestOpenAICompatibleMaxTokenFieldsOllamaDefault(t *testing.T) {
	tests := []struct {
		name            string
		maxTokens       int
		provider        string
		baseURL         string
		wantMaxTokens   int
		wantMaxCompletion int
	}{
		// Explicit max_tokens always passed through unchanged.
		{"explicit tokens cloud", 4096, "openai", "https://api.openai.com/v1", 4096, 0},
		{"explicit tokens ollama", 1000, "ollama", "http://localhost:11434/v1", 1000, 0},
		// Ollama/custom/local without max_tokens → 65536 floor.
		{"ollama no limit", 0, "ollama", "http://localhost:11434/v1", 65536, 0},
		{"custom no limit", 0, "custom", "http://localhost:8080/v1", 65536, 0},
		{"lmstudio no limit", 0, "lmstudio", "http://localhost:1234/v1", 65536, 0},
		{"local provider", 0, "local", "http://127.0.0.1:1234/v1", 65536, 0},
		{"vllm provider", 0, "vllm", "http://localhost:8000/v1", 65536, 0},
		// Cloud providers without explicit limit → no max_tokens sent.
		{"openai no limit", 0, "openai", "https://api.openai.com/v1", 0, 0},
		{"anthropic no limit", 0, "anthropic", "https://api.anthropic.com", 0, 0},
		{"openrouter no limit", 0, "openrouter", "https://openrouter.ai/api/v1", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMax, gotComp := openAICompatibleMaxTokenFields(tt.maxTokens, tt.provider, tt.baseURL, "any-model")
			if gotMax != tt.wantMaxTokens || gotComp != tt.wantMaxCompletion {
				t.Errorf("openAICompatibleMaxTokenFields(%d, %q, %q) = (%d, %d), want (%d, %d)",
					tt.maxTokens, tt.provider, tt.baseURL, gotMax, gotComp, tt.wantMaxTokens, tt.wantMaxCompletion)
			}
		})
	}
}
