package modelchoice

import (
	"reflect"
	"testing"
)

func TestSuggestionsForPromptTrimsDedupesAndBounds(t *testing.T) {
	got := SuggestionsForPrompt([]string{" alpha ", "ALPHA", "", "beta", "gamma"}, 2)
	want := []string{"alpha", "beta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SuggestionsForPrompt() = %#v, want %#v", got, want)
	}
}

func TestSuggestionsForPromptUnlimitedKeepsAllUnique(t *testing.T) {
	got := SuggestionsForPrompt([]string{"alpha", "beta", "ALPHA", "gamma"}, UnlimitedSuggestions)
	want := []string{"alpha", "beta", "gamma"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SuggestionsForPrompt() = %#v, want %#v", got, want)
	}
}

func TestDefaultChoiceIDAndIndexChoiceMatchCurrentCaseInsensitively(t *testing.T) {
	models := []string{"alpha", " Beta ", "gamma"}
	if got := DefaultChoiceID(models, "beta"); got != "Beta" {
		t.Fatalf("DefaultChoiceID() = %q, want Beta", got)
	}
	if got := IndexChoice(models, "BETA"); got != 1 {
		t.Fatalf("IndexChoice() = %d, want 1", got)
	}
}
