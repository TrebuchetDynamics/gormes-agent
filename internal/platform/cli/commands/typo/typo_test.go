package typo

import (
	"strings"
	"testing"
)

func TestTypoSuggestionLoginIsNoLongerATypo(t *testing.T) {
	// gormes login is now a real registered-but-hidden cobra command; the typo
	// handler must not intercept it so cobra can route to the real command.
	got, ok := TypoSuggestion([]string{"login"})
	if ok {
		t.Fatalf("TypoSuggestion(login) = %q, true; want no suggestion (login is a real command now)", got)
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
