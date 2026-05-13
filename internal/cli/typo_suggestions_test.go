package cli

import (
	"strings"
	"testing"
)

func TestTypoSuggestionDoesNotInterceptLoginCommand(t *testing.T) {
	if got, ok := TypoSuggestion([]string{"login"}); ok || got != "" {
		t.Fatalf("TypoSuggestion(login) = %q, %v; want no suggestion for registered login command", got, ok)
	}
}

func TestTypoSuggestionDoesNotInspectLoginProviderArgValues(t *testing.T) {
	got, ok := TypoSuggestion([]string{"login", "--provider", "plain-secret-provider", "--portal-url", "https://example.invalid"})
	if ok || got != "" {
		t.Fatalf("TypoSuggestion(login --provider ...) = %q, %v; want no suggestion for registered login command", got, ok)
	}
	if containsAny(got, "plain-secret-provider", "https://example.invalid") {
		t.Fatalf("suggestion leaked arg value: %q", got)
	}
}

func TestTypoSuggestionIgnoresNonLoginCommands(t *testing.T) {
	if got, ok := TypoSuggestion([]string{"logout"}); ok || got != "" {
		t.Fatalf("TypoSuggestion(logout) = %q, %v; want no suggestion", got, ok)
	}
}

func containsAny(s string, needles ...string) bool {
	for _, needle := range needles {
		if needle != "" && strings.Contains(s, needle) {
			return true
		}
	}
	return false
}
