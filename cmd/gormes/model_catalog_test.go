package main

import (
	"strings"
	"testing"
)

func TestModelOpenCodeSuggestionsUseCatalogFloor(t *testing.T) {
	got := defaultModelCatalogSuggestions("opencode-go")
	for _, want := range []string{"mimo-v2-pro", "kimi-k2.6"} {
		if !containsStringModelCatalog(got, want) {
			t.Fatalf("defaultModelCatalogSuggestions(opencode-go) missing %q: %#v", want, got)
		}
	}
}

func TestModelCatalogPromptShowsBoundedSuggestions(t *testing.T) {
	var out strings.Builder
	in := strings.NewReader("custom-model\n")
	got, err := promptModelChoice(in, &out, "opencode-go", "", []string{
		"mimo-v2-pro",
		"kimi-k2.6",
		"glm-5.1",
		"glm-5",
		"mimo-v2-omni",
		"minimax-m2.7",
	})
	if err != nil {
		t.Fatalf("promptModelChoice: %v", err)
	}
	if got != "custom-model" {
		t.Fatalf("model = %q, want custom-model", got)
	}
	text := out.String()
	if !strings.Contains(text, "Suggested models for opencode-go: mimo-v2-pro, kimi-k2.6, glm-5.1, glm-5, mimo-v2-omni") {
		t.Fatalf("prompt output missing bounded suggestions:\n%s", text)
	}
	if strings.Contains(text, "minimax-m2.7") {
		t.Fatalf("prompt output was not bounded:\n%s", text)
	}
}

func containsStringModelCatalog(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
