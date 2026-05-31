package cli

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/commands"
)

func TestTypoSuggestionFacadeDelegatesToCommandsPackage(t *testing.T) {
	args := []string{"login", "--provider", "plain-secret-provider", "--portal-url", "https://example.invalid"}
	got, gotOK := TypoSuggestion(args)
	want, wantOK := commands.TypoSuggestion(args)
	if got != want || gotOK != wantOK {
		t.Fatalf("TypoSuggestion facade = %q, %v; want commands package result %q, %v", got, gotOK, want, wantOK)
	}
}
