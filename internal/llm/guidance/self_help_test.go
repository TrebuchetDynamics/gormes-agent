package guidance

import (
	"strings"
	"testing"
)

func TestGormesSelfHelpGuidanceFacade(t *testing.T) {
	guidance, ok := GormesSelfHelpGuidanceForPrompt("How do I configure Gormes Agent?")
	if !ok {
		t.Fatal("GormesSelfHelpGuidanceForPrompt() ok = false, want true")
	}
	for _, want := range []string{"Gormes", "https://docs.gormes.ai/", "self-help-unavailable"} {
		if !strings.Contains(guidance, want) {
			t.Fatalf("guidance missing %q:\n%s", want, guidance)
		}
	}
}
