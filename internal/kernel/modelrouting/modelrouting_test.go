package modelrouting

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestRouteNormalizesImplementedProviderManifest(t *testing.T) {
	route, ok := Route(" openrouter ", " openai/gpt-4o-mini ")
	if !ok {
		t.Fatal("Route ok = false, want true for implemented provider")
	}
	if route.Provider != "openrouter" {
		t.Fatalf("Provider = %q, want openrouter", route.Provider)
	}
	if route.Model != "openai/gpt-4o-mini" {
		t.Fatalf("Model = %q, want trimmed model", route.Model)
	}
	if route.APIMode != "chat_completions" {
		t.Fatalf("APIMode = %q, want chat_completions", route.APIMode)
	}
	if route.APIKeyEnv == "" {
		t.Fatal("APIKeyEnv is empty, want first configured credential env")
	}
}

func TestTurnSelectionPrefersTrimmedOverride(t *testing.T) {
	if got := SelectTurnModel("resident", " override "); got != "override" {
		t.Fatalf("SelectTurnModel = %q, want override", got)
	}
	status := llm.ProviderStatus{Runtime: "chat_completions"}
	got := SelectTurnReasoningEffort("low", " high ", status)
	if got.Source != llm.ReasoningEffortSourceTurnOverride || got.Effort != "high" {
		t.Fatalf("SelectTurnReasoningEffort = %+v, want turn override high", got)
	}
}

func TestSharedStringHelpersTrimInputs(t *testing.T) {
	if got := FirstNonEmpty(" ", " configured "); got != "configured" {
		t.Fatalf("FirstNonEmpty = %q, want configured", got)
	}
	if !MatchesAny("openai/gpt-4o", []string{" gemini ", " GPT "}) {
		t.Fatal("MatchesAny did not match trimmed lowercase needle")
	}
	if !ShouldSwapProvider(" openai ", "fallback", " anthropic ") {
		t.Fatal("ShouldSwapProvider = false, want true for different non-empty provider")
	}
	if ShouldSwapProvider(" openai ", "fallback", " ") {
		t.Fatal("ShouldSwapProvider = true, want false for empty next provider")
	}
}
