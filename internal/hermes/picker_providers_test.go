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
