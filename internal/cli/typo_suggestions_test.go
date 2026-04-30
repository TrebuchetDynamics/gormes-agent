package cli

import (
	"strings"
	"testing"
)

func TestTypoSuggestionLoginRedirectsToAuthAddOAuth(t *testing.T) {
	got, ok := TypoSuggestion([]string{"login"})
	if !ok {
		t.Fatalf("TypoSuggestion(login) ok = false")
	}
	want := "did you mean \"gormes auth add <provider> --type oauth\"?"
	if got != want {
		t.Fatalf("TypoSuggestion(login) = %q, want %q", got, want)
	}
}

func TestTypoSuggestionLoginRedactsProviderArgValues(t *testing.T) {
	got, ok := TypoSuggestion([]string{"login", "--provider", "plain-secret-provider", "--portal-url", "https://example.invalid"})
	if !ok {
		t.Fatalf("TypoSuggestion(login --provider ...) ok = false")
	}
	if got != "did you mean \"gormes auth add <provider> --type oauth\"?" {
		t.Fatalf("TypoSuggestion with provider = %q", got)
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
