package hermes

import "testing"

func TestListPickerProvidersCarriesModelLists(t *testing.T) {
	providers := ListPickerProviders()

	openAICodex, ok := findPickerProviderForTest(providers, "openai-codex")
	if !ok {
		t.Fatalf("ListPickerProviders() missing openai-codex provider")
	}
	if len(openAICodex.Models) <= 5 {
		t.Fatalf("openai-codex models = %d, want more than static prompt ceiling: %#v", len(openAICodex.Models), openAICodex.Models)
	}
	if !containsPickerModelForTest(openAICodex.Models, "gpt-5.2-codex") {
		t.Fatalf("openai-codex models missing gpt-5.2-codex: %#v", openAICodex.Models)
	}

	openRouter, ok := findPickerProviderForTest(providers, "openrouter")
	if !ok {
		t.Fatalf("ListPickerProviders() missing openrouter provider")
	}
	if len(openRouter.Models) < 20 {
		t.Fatalf("openrouter models = %d, want full curated fallback list: %#v", len(openRouter.Models), openRouter.Models)
	}
	for _, want := range []string{"anthropic/claude-opus-4.7", "moonshotai/kimi-k2.6", "openai/gpt-5.5", "tencent/hy3-preview:free"} {
		if !containsPickerModelForTest(openRouter.Models, want) {
			t.Fatalf("openrouter models missing %s: %#v", want, openRouter.Models)
		}
	}

	opencodeGo, ok := findPickerProviderForTest(providers, "opencode-go")
	if !ok {
		t.Fatalf("ListPickerProviders() missing opencode-go provider")
	}
	if !containsPickerModelForTest(opencodeGo.Models, "kimi-k2.6") {
		t.Fatalf("opencode-go models missing kimi-k2.6: %#v", opencodeGo.Models)
	}
}

func findPickerProviderForTest(providers []PickerProvider, slug string) (PickerProvider, bool) {
	for _, provider := range providers {
		if provider.Slug == slug {
			return provider, true
		}
	}
	return PickerProvider{}, false
}

func containsPickerModelForTest(models []string, want string) bool {
	for _, model := range models {
		if model == want {
			return true
		}
	}
	return false
}
