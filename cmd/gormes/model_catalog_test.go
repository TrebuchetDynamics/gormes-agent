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

func TestModelCatalogPromptShowsBoundedSelectableList(t *testing.T) {
	var out strings.Builder
	in := strings.NewReader("2\n")
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
	if got != "kimi-k2.6" {
		t.Fatalf("model = %q, want kimi-k2.6", got)
	}
	text := out.String()
	for _, want := range []string{
		"Select model for opencode-go:",
		"1. mimo-v2-pro",
		"2. kimi-k2.6",
		"5. mimo-v2-omni",
		"Choice [1-5], custom model, or q to cancel:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("prompt output missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "Suggested models for opencode-go:") {
		t.Fatalf("prompt output still uses comma-separated suggestions:\n%s", text)
	}
	if strings.Contains(text, "minimax-m2.7") {
		t.Fatalf("prompt output was not bounded:\n%s", text)
	}
}

func TestModelCatalogPromptAcceptsCustomModel(t *testing.T) {
	var out strings.Builder
	in := strings.NewReader("custom-model\n")
	got, err := promptModelChoice(in, &out, "opencode-go", "", []string{"kimi-k2.6"})
	if err != nil {
		t.Fatalf("promptModelChoice: %v", err)
	}
	if got != "custom-model" {
		t.Fatalf("model = %q, want custom-model", got)
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
