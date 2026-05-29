package llm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProviderDefaultModel_OpenAICodexPrefersLocalCodexDefault(t *testing.T) {
	codexHome := t.TempDir()
	if err := os.WriteFile(filepath.Join(codexHome, "config.toml"), []byte(`model = "gpt-5.4-mini"`), 0o600); err != nil {
		t.Fatalf("write codex config: %v", err)
	}

	got := ResolveProviderDefaultModel("openai-codex", ProviderDefaultModelOptions{CodexHome: codexHome})
	if got.Model != "gpt-5.4-mini" || got.Source != ProviderDefaultModelSourceCodexConfig {
		t.Fatalf("ResolveProviderDefaultModel = %#v, want gpt-5.4-mini from codex config", got)
	}
}

func TestProviderDefaultModel_OpenAICodexUsesLocalCachePriority(t *testing.T) {
	codexHome := t.TempDir()
	cache := `{
  "models": [
    {"slug": "hidden-model", "priority": 0, "visibility": "hidden"},
    {"slug": "account-visible-model", "priority": 2},
    {"slug": "cli-only-visible-model", "priority": 7, "supported_in_api": false},
    {"slug": "later-visible-model", "priority": 7}
  ]
}`
	if err := os.WriteFile(filepath.Join(codexHome, "models_cache.json"), []byte(cache), 0o600); err != nil {
		t.Fatalf("write codex cache: %v", err)
	}

	got := ResolveProviderDefaultModel("openai-codex", ProviderDefaultModelOptions{CodexHome: codexHome})
	if got.Model != "account-visible-model" || got.Source != ProviderDefaultModelSourceCodexCache {
		t.Fatalf("ResolveProviderDefaultModel = %#v, want first visible cache model", got)
	}
}

func TestProviderDefaultModel_OpenAICodexKeepsCLIOnlyCacheModel(t *testing.T) {
	codexHome := t.TempDir()
	cache := `{
  "models": [
    {"slug": "gpt-5-hidden-codex", "priority": 0, "visibility": "hidden"},
    {"slug": "gpt-5.3-codex-spark", "priority": 1, "supported_in_api": false},
    {"slug": "gpt-5.4", "priority": 2, "supported_in_api": true}
  ]
}`
	if err := os.WriteFile(filepath.Join(codexHome, "models_cache.json"), []byte(cache), 0o600); err != nil {
		t.Fatalf("write codex cache: %v", err)
	}

	got := ResolveProviderDefaultModel("openai-codex", ProviderDefaultModelOptions{CodexHome: codexHome})
	if got.Model != "gpt-5.3-codex-spark" || got.Source != ProviderDefaultModelSourceCodexCache {
		t.Fatalf("ResolveProviderDefaultModel = %#v, want CLI-only Spark cache model", got)
	}
}

func TestProviderDefaultModel_OpenAICodexFallsBackToCuratedCatalog(t *testing.T) {
	got := ResolveProviderDefaultModel("openai-codex", ProviderDefaultModelOptions{CodexHome: t.TempDir()})
	if got.Model != "gpt-5.5" || got.Source != ProviderDefaultModelSourceCuratedFallback {
		t.Fatalf("ResolveProviderDefaultModel = %#v, want curated gpt-5.5 fallback", got)
	}
}

func TestProviderDefaultModel_OpenRouterUsesFreeModelFallback(t *testing.T) {
	got := ResolveProviderDefaultModel("openrouter-free", ProviderDefaultModelOptions{})
	if got.Provider != "openrouter" || got.Model != "deepseek/deepseek-chat-v3-0324:free" || got.Source != ProviderDefaultModelSourceStaticCatalog {
		t.Fatalf("ResolveProviderDefaultModel(openrouter-free) = %#v, want OpenRouter free DeepSeek V3 fallback via canonical openrouter provider", got)
	}
}

func TestProviderDefaultModel_GroqUsesFreeTierFriendlyFallback(t *testing.T) {
	got := ResolveProviderDefaultModel("groq", ProviderDefaultModelOptions{})
	if got.Model != "llama-3.3-70b-versatile" || got.Source != ProviderDefaultModelSourceStaticCatalog {
		t.Fatalf("ResolveProviderDefaultModel(groq) = %#v, want Groq llama-3.3-70b free-tier fallback", got)
	}
}

func TestProviderDefaultModel_GoogleAIStudioAliasUsesGeminiFlash(t *testing.T) {
	got := ResolveProviderDefaultModel("google-ai-studio", ProviderDefaultModelOptions{})
	if got.Provider != "gemini" || got.Model != "gemini-2.5-flash" || got.Source != ProviderDefaultModelSourceStaticCatalog {
		t.Fatalf("ResolveProviderDefaultModel(google-ai-studio) = %#v, want Gemini Flash via canonical gemini provider", got)
	}
}
