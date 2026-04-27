package hermes

import "testing"

func TestAzureFoundryAPIMode_ResponsesFamilies(t *testing.T) {
	for _, model := range []string{
		"gpt-5",
		"gpt-5.3-codex",
		"gpt-5-mini",
		"codex",
		"codex-mini",
		"o1-preview",
		"o3-mini",
		"o4-mini",
	} {
		t.Run(model, func(t *testing.T) {
			got, ok := AzureFoundryAPIModeForModel(model)
			if !ok {
				t.Fatalf("AzureFoundryAPIModeForModel(%q) returned no override, want %q", model, AzureTransportCodexResponses)
			}
			if got != AzureTransportCodexResponses {
				t.Fatalf("AzureFoundryAPIModeForModel(%q) = %q, want %q", model, got, AzureTransportCodexResponses)
			}
		})
	}
}

func TestAzureFoundryAPIMode_NonResponsesFamilies(t *testing.T) {
	for _, model := range []string{
		"gpt-4o",
		"gpt-4.1",
		"gpt-3.5-turbo",
		"llama-3.1",
		"mistral-large",
		"grok-4",
		"",
		"   ",
	} {
		t.Run(model, func(t *testing.T) {
			got, ok := AzureFoundryAPIModeForModel(model)
			if ok || got != "" {
				t.Fatalf("AzureFoundryAPIModeForModel(%q) = (%q, %v), want no override", model, got, ok)
			}
		})
	}
}

func TestAzureFoundryAPIMode_StripsVendorPrefixAndCase(t *testing.T) {
	got, ok := AzureFoundryAPIModeForModel("openai/GPT-5.3-Codex")
	if !ok {
		t.Fatalf("AzureFoundryAPIModeForModel(openai/GPT-5.3-Codex) returned no override, want %q", AzureTransportCodexResponses)
	}
	if got != AzureTransportCodexResponses {
		t.Fatalf("AzureFoundryAPIModeForModel(openai/GPT-5.3-Codex) = %q, want %q", got, AzureTransportCodexResponses)
	}

	got, ok = AzureFoundryAPIModeForModel("openai/gpt-4o")
	if ok || got != "" {
		t.Fatalf("AzureFoundryAPIModeForModel(openai/gpt-4o) = (%q, %v), want no override", got, ok)
	}
}
