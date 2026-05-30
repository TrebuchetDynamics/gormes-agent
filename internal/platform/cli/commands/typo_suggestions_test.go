package commands

import (
	"strings"
	"testing"
)

func TestTypoSuggestionGuidesRemovedLoginCommand(t *testing.T) {
	got, ok := TypoSuggestion([]string{"login"})
	if !ok || !strings.Contains(got, "gormes auth add <provider> --type oauth") {
		t.Fatalf("TypoSuggestion(login) = %q, %v; want auth add guidance", got, ok)
	}
}

func TestTypoSuggestionDoesNotInspectLoginProviderArgValues(t *testing.T) {
	got, ok := TypoSuggestion([]string{"login", "--provider", "plain-secret-provider", "--portal-url", "https://example.invalid"})
	if !ok || !strings.Contains(got, "gormes auth add <provider> --type oauth") {
		t.Fatalf("TypoSuggestion(login --provider ...) = %q, %v; want auth add guidance", got, ok)
	}
	if containsAny(got, "plain-secret-provider", "https://example.invalid") {
		t.Fatalf("suggestion leaked arg value: %q", got)
	}
}

func TestTypoSuggestionGuidesRemovedOnboardCommand(t *testing.T) {
	got, ok := TypoSuggestion([]string{"onboard", "--json"})
	if !ok || !strings.Contains(got, "gormes setup") || !strings.Contains(got, "gormes doctor --offline --target terminal --json") {
		t.Fatalf("TypoSuggestion(onboard) = %q, %v; want setup/doctor guidance", got, ok)
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
