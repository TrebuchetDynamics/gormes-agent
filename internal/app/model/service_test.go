package model

import (
	"strings"
	"testing"
)

func TestPromptModelChoiceTextUsesBoundedNumberSelection(t *testing.T) {
	var out strings.Builder
	got, err := PromptModelChoiceText(strings.NewReader("2\n"), &out, "opencode-go", "", []string{"mimo-v2-pro", "kimi-k2.6"})
	if err != nil {
		t.Fatalf("PromptModelChoiceText: %v", err)
	}
	if got != "kimi-k2.6" {
		t.Fatalf("model = %q, want kimi-k2.6", got)
	}
	if !strings.Contains(out.String(), "Choice [1-2], custom model, or q to cancel:") {
		t.Fatalf("prompt missing choice contract:\n%s", out.String())
	}
}

func TestModelCatalogSuggestionsForPromptUnlimited(t *testing.T) {
	values := []string{"a", "b", "c", "d", "e", "f"}
	got := ModelCatalogSuggestionsForPrompt(values, SuggestionLimitUnlimited)
	if len(got) != len(values) {
		t.Fatalf("suggestions = %v, want all %v", got, values)
	}
}
