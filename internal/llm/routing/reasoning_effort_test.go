package routing

import (
	"strings"
	"testing"
)

func TestResolveReasoningEffort(t *testing.T) {
	for _, value := range []string{"none", "minimal", "low", "medium", "high", "xhigh"} {
		t.Run(value, func(t *testing.T) {
			evidence := ResolveReasoningEffort(value, ReasoningEffortSourceTurnOverride, ProviderStatus{Runtime: "chat_completions"})
			if !evidence.Forwarded {
				t.Fatalf("Forwarded = false, want true for %q: %+v", value, evidence)
			}
			if evidence.Effort != ReasoningEffort(value) {
				t.Fatalf("Effort = %q, want %q", evidence.Effort, value)
			}
		})
	}
}

func TestResolveReasoningEffortDefaultsAndErrors(t *testing.T) {
	defaultEvidence := ResolveReasoningEffort("", ReasoningEffortSourceConfigDefault, ProviderStatus{Runtime: "chat_completions"})
	if defaultEvidence.State != ReasoningEffortStateDefault || defaultEvidence.Forwarded {
		t.Fatalf("default evidence = %+v, want default and not forwarded", defaultEvidence)
	}

	invalid := ResolveReasoningEffort("max", ReasoningEffortSourceTurnOverride, ProviderStatus{Runtime: "chat_completions"})
	if invalid.State != ReasoningEffortStateInvalid || invalid.Forwarded {
		t.Fatalf("invalid evidence = %+v, want invalid and not forwarded", invalid)
	}

	unsupported := ResolveReasoningEffort("high", ReasoningEffortSourceTurnOverride, ProviderStatus{Runtime: "anthropic_messages"})
	if unsupported.State != ReasoningEffortStateUnsupported || unsupported.Forwarded {
		t.Fatalf("unsupported evidence = %+v, want unsupported and not forwarded", unsupported)
	}
	if !strings.Contains(unsupported.Reason, "anthropic_messages") {
		t.Fatalf("Reason = %q, want runtime evidence", unsupported.Reason)
	}
}
