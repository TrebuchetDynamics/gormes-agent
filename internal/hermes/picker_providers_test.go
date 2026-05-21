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
	for _, want := range []string{
		"deepseek/deepseek-r1:free",
		"deepseek/deepseek-chat-v3-0324:free",
		"meta-llama/llama-4-maverick:free",
		"qwen/qwen3-235b-a22b:free",
	} {
		if !containsPickerModelForTest(openRouter.Models, want) {
			t.Fatalf("openrouter free-model catalog missing %s: %#v", want, openRouter.Models)
		}
	}

	opencodeGo, ok := findPickerProviderForTest(providers, "opencode-go")
	if !ok {
		t.Fatalf("ListPickerProviders() missing opencode-go provider")
	}
	if !containsPickerModelForTest(opencodeGo.Models, "kimi-k2.6") {
		t.Fatalf("opencode-go models missing kimi-k2.6: %#v", opencodeGo.Models)
	}

	groq, ok := findPickerProviderForTest(providers, "groq")
	if !ok {
		t.Fatalf("ListPickerProviders() missing groq provider for free fallback setup")
	}
	if !containsPickerModelForTest(groq.Models, "llama-3.3-70b-versatile") {
		t.Fatalf("groq models missing llama-3.3-70b-versatile: %#v", groq.Models)
	}

	gemini, ok := findPickerProviderForTest(providers, "gemini")
	if !ok {
		t.Fatalf("ListPickerProviders() missing gemini provider for Google AI Studio fallback setup")
	}
	if !containsPickerModelForTest(gemini.Models, "gemini-2.5-flash") {
		t.Fatalf("gemini models missing gemini-2.5-flash: %#v", gemini.Models)
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
