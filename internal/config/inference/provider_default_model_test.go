package inference

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveProviderDefaultOpenAICodexPlaceholderUsesCuratedFallback(t *testing.T) {
	setupProviderDefaultModelEnv(t)

	provider, model, source, resolved := ResolveProviderDefault("openai-codex", "hermes-agent")
	if !resolved {
		t.Fatal("ResolveProviderDefault resolved = false, want true")
	}
	if provider != "openai-codex" || model != "gpt-5.5" {
		t.Fatalf("provider/model = %q/%q, want openai-codex/gpt-5.5", provider, model)
	}
	if source != "curated_fallback" {
		t.Fatalf("source = %q, want curated_fallback", source)
	}
}

func TestResolveProviderDefaultOpenAICodexEmptyModelUsesCodexConfig(t *testing.T) {
	setupProviderDefaultModelEnv(t)
	codexHome := os.Getenv("CODEX_HOME")
	if err := os.WriteFile(filepath.Join(codexHome, "config.toml"), []byte(`model = "gpt-5.4-mini"`), 0o600); err != nil {
		t.Fatalf("write codex config: %v", err)
	}

	_, model, source, resolved := ResolveProviderDefault("openai-codex", "")
	if !resolved {
		t.Fatal("ResolveProviderDefault resolved = false, want true")
	}
	if model != "gpt-5.4-mini" {
		t.Fatalf("model = %q, want Codex config model", model)
	}
	if source != "codex_config" {
		t.Fatalf("source = %q, want codex_config", source)
	}
}

func TestResolveProviderDefaultPreservesExplicitOperatorModel(t *testing.T) {
	setupProviderDefaultModelEnv(t)

	provider, model, source, resolved := ResolveProviderDefault("openai-codex", "gpt-5.3-codex")
	if resolved {
		t.Fatal("ResolveProviderDefault resolved = true, want false for explicit model")
	}
	if provider != "openai-codex" || model != "gpt-5.3-codex" {
		t.Fatalf("provider/model = %q/%q, want explicit operator values", provider, model)
	}
	if source != "explicit_operator_config" {
		t.Fatalf("source = %q, want explicit_operator_config", source)
	}
}

func setupProviderDefaultModelEnv(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("GORMES_HOME", filepath.Join(root, "gormes"))
	t.Setenv("HERMES_HOME", filepath.Join(root, "hermes"))
	t.Setenv("CODEX_HOME", filepath.Join(root, "codex"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg-config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg-data"))
	if err := os.MkdirAll(os.Getenv("CODEX_HOME"), 0o755); err != nil {
		t.Fatalf("create CODEX_HOME: %v", err)
	}
}
