package identity

import "testing"

func TestProviderNormalizesSharedAliases(t *testing.T) {
	tests := map[string]string{
		" codex ":          "openai-codex",
		"github-copilot":   "copilot",
		"google-ai-studio": "gemini",
		"open-router":      "openrouter",
		"ollama_cloud":     "ollama-cloud",
	}
	for input, want := range tests {
		if got := Provider(input); got != want {
			t.Fatalf("Provider(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestModelBaseStripsProviderAndVariant(t *testing.T) {
	if got := ModelBase("anthropic/Claude-Opus-4.6:beta"); got != "claude-opus-4.6" {
		t.Fatalf("ModelBase() = %q", got)
	}
}
