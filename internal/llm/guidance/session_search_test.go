package guidance

import (
	"strings"
	"testing"
)

func TestSessionSearchGuidance_ByteEquivalent(t *testing.T) {
	if SessionSearchGuidance == "" {
		t.Fatal("SessionSearchGuidance constant must not be empty")
	}
	if !strings.Contains(SessionSearchGuidance, "session_search") {
		t.Fatal("SessionSearchGuidance must contain 'session_search'")
	}
}

func TestSessionSearchGuidance_SwitchContract(t *testing.T) {
	assertGuidanceSwitch(t, guidanceSwitchCase{
		name:               "session_search",
		guidance:           SessionSearchGuidance,
		build:              BuildSessionSearchGuidance,
		injectedEvidence:   "session_search_guidance_injected",
		suppressedEvidence: "session_search_guidance_suppressed",
	})
}
