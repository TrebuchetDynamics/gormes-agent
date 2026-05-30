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

func TestSessionSearchGuidance_InjectedWhenEnabled(t *testing.T) {
	result := BuildSessionSearchGuidance(true)
	if !result.Injected {
		t.Fatal("expected injected when enabled")
	}
	if result.Guidance != SessionSearchGuidance {
		t.Fatal("expected full guidance text")
	}
	if result.Evidence != "session_search_guidance_injected" {
		t.Fatalf("expected evidence=session_search_guidance_injected, got %s", result.Evidence)
	}
}

func TestSessionSearchGuidance_SuppressedWhenDisabled(t *testing.T) {
	result := BuildSessionSearchGuidance(false)
	if result.Injected {
		t.Fatal("expected not injected when disabled")
	}
	if result.Guidance != "" {
		t.Fatal("expected empty guidance when suppressed")
	}
	if !strings.Contains(result.Evidence, "session_search_guidance_suppressed") {
		t.Fatalf("expected suppression evidence, got %s", result.Evidence)
	}
}

func TestSessionSearchGuidance_Pure(t *testing.T) {
	r1 := BuildSessionSearchGuidance(true)
	r2 := BuildSessionSearchGuidance(true)
	if r1.Guidance != r2.Guidance {
		t.Fatal("BuildSessionSearchGuidance must be deterministic")
	}
}
