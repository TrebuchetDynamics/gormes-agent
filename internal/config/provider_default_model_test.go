package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProviderDefaultModelResolved_OpenAICodexPlaceholderUsesCuratedFallback(t *testing.T) {
	root := setupProviderDefaultModelConfigEnv(t)
	writeProviderDefaultModelConfig(t, root, "[hermes]\nprovider = 'openai-codex'\nmodel = 'hermes-agent'\n")

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load(nil): %v", err)
	}
	if cfg.Hermes.Model != "gpt-5.5" {
		t.Fatalf("Hermes.Model = %q, want curated openai-codex fallback", cfg.Hermes.Model)
	}
	if cfg.Hermes.ModelResolutionSource != "curated_fallback" {
		t.Fatalf("ModelResolutionSource = %q, want curated_fallback", cfg.Hermes.ModelResolutionSource)
	}
}

func TestProviderDefaultModelResolved_OpenAICodexEmptyModelUsesCodexConfig(t *testing.T) {
	root := setupProviderDefaultModelConfigEnv(t)
	writeProviderDefaultModelConfig(t, root, "[hermes]\nprovider = 'openai-codex'\n")
	codexHome := os.Getenv("CODEX_HOME")
	if err := os.WriteFile(filepath.Join(codexHome, "config.toml"), []byte(`model = "gpt-5.4-mini"`), 0o600); err != nil {
		t.Fatalf("write codex config: %v", err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load(nil): %v", err)
	}
	if cfg.Hermes.Model != "gpt-5.4-mini" {
		t.Fatalf("Hermes.Model = %q, want Codex config model", cfg.Hermes.Model)
	}
	if cfg.Hermes.ModelResolutionSource != "codex_config" {
		t.Fatalf("ModelResolutionSource = %q, want codex_config", cfg.Hermes.ModelResolutionSource)
	}
}

func TestProviderDefaultModelResolved_PreservesExplicitOperatorModel(t *testing.T) {
	root := setupProviderDefaultModelConfigEnv(t)
	writeProviderDefaultModelConfig(t, root, "[hermes]\nprovider = 'openai-codex'\nmodel = 'gpt-5.3-codex'\n")

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load(nil): %v", err)
	}
	if cfg.Hermes.Model != "gpt-5.3-codex" {
		t.Fatalf("Hermes.Model = %q, want explicit operator model", cfg.Hermes.Model)
	}
	if cfg.Hermes.ModelResolutionSource != "explicit_operator_config" {
		t.Fatalf("ModelResolutionSource = %q, want explicit_operator_config", cfg.Hermes.ModelResolutionSource)
	}
}

func setupProviderDefaultModelConfigEnv(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("GORMES_HOME", filepath.Join(root, "gormes"))
	t.Setenv("HERMES_HOME", filepath.Join(root, "hermes"))
	t.Setenv("CODEX_HOME", filepath.Join(root, "codex"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg-config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg-data"))
	t.Setenv("GORMES_ENDPOINT", "")
	t.Setenv("GORMES_MODEL", "")
	t.Setenv("GORMES_API_KEY", "")
	t.Setenv("GORMES_INFERENCE_MODEL", "")
	t.Setenv("GORMES_INFERENCE_PROVIDER", "")
	if err := os.MkdirAll(os.Getenv("CODEX_HOME"), 0o755); err != nil {
		t.Fatalf("create CODEX_HOME: %v", err)
	}
	return root
}

func writeProviderDefaultModelConfig(t *testing.T, root, body string) {
	t.Helper()
	path := filepath.Join(root, "gormes", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
