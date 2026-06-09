package llm

import (
	"strings"
	"testing"
)

func TestAgentIdentityForProfileUsesNamedProfileAsAssistantName(t *testing.T) {
	got := AgentIdentityForProfile("mineru")
	for _, want := range []string{"You are Mineru", "If asked your name, answer Mineru", "run by gormes"} {
		if !strings.Contains(got, want) {
			t.Fatalf("AgentIdentityForProfile(mineru) missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "You are Gorm") {
		t.Fatalf("named profile identity must not reassert Gorm:\n%s", got)
	}
}

func TestAgentIdentityForProfileKeepsDefaultPersonaForMainProfiles(t *testing.T) {
	for _, profile := range []string{"", "default", "main", "root"} {
		if got := AgentIdentityForProfile(profile); got != DefaultAgentIdentity {
			t.Fatalf("AgentIdentityForProfile(%q) = %q, want default identity", profile, got)
		}
	}
}
