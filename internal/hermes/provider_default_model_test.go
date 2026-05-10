package hermes

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
    {"slug": "unsupported-model", "priority": 1, "supported_in_api": false},
    {"slug": "account-visible-model", "priority": 2},
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

func TestProviderDefaultModel_OpenAICodexFallsBackToCuratedCatalog(t *testing.T) {
	got := ResolveProviderDefaultModel("openai-codex", ProviderDefaultModelOptions{CodexHome: t.TempDir()})
	if got.Model != "gpt-5.5" || got.Source != ProviderDefaultModelSourceCuratedFallback {
		t.Fatalf("ResolveProviderDefaultModel = %#v, want curated gpt-5.5 fallback", got)
	}
}
