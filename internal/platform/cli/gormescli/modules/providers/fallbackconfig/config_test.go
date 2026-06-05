package fallbackconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppendAndWriteChainPreserveFallbackConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("[hermes]\nprovider = 'openai-codex'\nmodel = 'gpt-5.5'\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	added, err := Append(path, Entry{Provider: "openrouter", Model: "anthropic/claude-sonnet-4.5", BaseURL: "https://openrouter.ai/api/v1"})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	if !added {
		t.Fatalf("Append added = false, want true")
	}
	added, err = Append(path, Entry{Provider: "openrouter", Model: "anthropic/claude-sonnet-4.5"})
	if err != nil {
		t.Fatalf("Append duplicate: %v", err)
	}
	if added {
		t.Fatalf("Append duplicate added = true, want false")
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Primary.Provider != "openai-codex" || cfg.Primary.Model != "gpt-5.5" {
		t.Fatalf("primary = %#v", cfg.Primary)
	}
	if len(cfg.Chain) != 1 || cfg.Chain[0].Provider != "openrouter" || cfg.Chain[0].Model != "anthropic/claude-sonnet-4.5" {
		t.Fatalf("chain after append = %#v", cfg.Chain)
	}

	if err := WriteChain(path, []Entry{{Provider: "ollama-cloud", Model: "qwen3-coder:480b"}}); err != nil {
		t.Fatalf("WriteChain: %v", err)
	}
	cfg, err = Load(path)
	if err != nil {
		t.Fatalf("Load after WriteChain: %v", err)
	}
	if len(cfg.Chain) != 1 || cfg.Chain[0].Provider != "ollama-cloud" || cfg.Chain[0].Model != "qwen3-coder:480b" {
		t.Fatalf("chain after write = %#v", cfg.Chain)
	}
}
