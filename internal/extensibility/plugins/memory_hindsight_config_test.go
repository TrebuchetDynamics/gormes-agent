package plugins

import (
	"testing"
)

func TestHindsightSetupConfigPatch_BlankInputsPreserveExistingLocalEmbeddedConfig(t *testing.T) {
	existing := HindsightConfig{
		Mode:        "local_embedded",
		LLMProvider: "openai_compatible",
		LLMBaseURL:  "https://api.test/v1",
		LLMModel:    "gpt-4o",
		Timeout:     120,
		IdleTimeout: 0,
		Extra:       map[string]string{"vendor_key": "abc-def"},
	}
	incoming := HindsightConfig{} // all blank

	result := PatchHindsightConfig(existing, incoming)

	if result.Mode != "local_embedded" {
		t.Fatalf("Mode = %q, want local_embedded", result.Mode)
	}
	if result.LLMProvider != "openai_compatible" {
		t.Fatalf("LLMProvider = %q, want openai_compatible", result.LLMProvider)
	}
	if result.LLMBaseURL != "https://api.test/v1" {
		t.Fatalf("LLMBaseURL = %q", result.LLMBaseURL)
	}
	if result.LLMModel != "gpt-4o" {
		t.Fatalf("LLMModel = %q", result.LLMModel)
	}
	if result.Timeout != 120 {
		t.Fatalf("Timeout = %d, want 120", result.Timeout)
	}
	if result.IdleTimeout != 0 {
		t.Fatalf("IdleTimeout = %d, want 0", result.IdleTimeout)
	}
	if result.Extra["vendor_key"] != "abc-def" {
		t.Fatalf("Extra vendor_key = %q", result.Extra["vendor_key"])
	}
}

func TestHindsightSetupConfigPatch_BlankAPIKeyPreservesReference(t *testing.T) {
	existing := HindsightConfig{APIKey: "sk-ref-12345"}
	incoming := HindsightConfig{APIKey: ""}
	result := PatchHindsightConfig(existing, incoming)
	if result.APIKey != "sk-ref-12345" {
		t.Fatalf("APIKey = %q, want preserved reference", result.APIKey)
	}
}

func TestHindsightSetupConfigPatch_ModeDefaultUsesExistingValue(t *testing.T) {
	existing := HindsightConfig{Mode: "ollama"}
	incoming := HindsightConfig{Mode: "invalid_mode"}
	result := PatchHindsightConfig(existing, incoming)
	if result.Mode != "ollama" {
		t.Fatalf("Mode = %q, want ollama (preserved)", result.Mode)
	}
}

func TestHindsightSetupConfigPatch_AppliesValidNewValues(t *testing.T) {
	existing := HindsightConfig{Mode: "local_embedded", Timeout: 60}
	incoming := HindsightConfig{Mode: "openai_compatible", LLMModel: "claude-3", Timeout: 90, APIKey: "new-key"}
	result := PatchHindsightConfig(existing, incoming)
	if result.Mode != "openai_compatible" {
		t.Fatalf("Mode = %q", result.Mode)
	}
	if result.LLMModel != "claude-3" {
		t.Fatalf("LLMModel = %q", result.LLMModel)
	}
	if result.Timeout != 90 {
		t.Fatalf("Timeout = %d", result.Timeout)
	}
	if result.APIKey != "new-key" {
		t.Fatalf("APIKey = %q", result.APIKey)
	}
}
