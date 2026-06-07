package guidance

import (
	"strings"
	"testing"
)

type guidanceSwitchCase struct {
	name               string
	guidance           string
	build              func(bool) GuidanceSwitchResult
	injectedEvidence   string
	suppressedEvidence string
}

func assertGuidanceSwitch(t *testing.T, tc guidanceSwitchCase) {
	t.Helper()

	injected := tc.build(true)
	if !injected.Injected {
		t.Fatalf("%s: expected injected when enabled", tc.name)
	}
	if injected.Guidance != tc.guidance {
		t.Fatalf("%s: expected full guidance text", tc.name)
	}
	if injected.Evidence != tc.injectedEvidence {
		t.Fatalf("%s: expected evidence=%s, got %s", tc.name, tc.injectedEvidence, injected.Evidence)
	}

	suppressed := tc.build(false)
	if suppressed.Injected {
		t.Fatalf("%s: expected not injected when disabled", tc.name)
	}
	if suppressed.Guidance != "" {
		t.Fatalf("%s: expected empty guidance when suppressed", tc.name)
	}
	if !strings.Contains(suppressed.Evidence, tc.suppressedEvidence) {
		t.Fatalf("%s: expected suppression evidence containing %q, got %s", tc.name, tc.suppressedEvidence, suppressed.Evidence)
	}

	again := tc.build(true)
	if injected.Guidance != again.Guidance || injected.Injected != again.Injected || injected.Evidence != again.Evidence {
		t.Fatalf("%s: expected deterministic guidance switch result", tc.name)
	}
}

func TestMemoryGuidance_ByteEquivalent(t *testing.T) {
	if MemoryGuidance == "" {
		t.Fatal("MemoryGuidance constant must not be empty")
	}
	if !strings.Contains(MemoryGuidance, "persistent memory") {
		t.Fatal("MemoryGuidance must contain 'persistent memory'")
	}
	if !strings.Contains(MemoryGuidance, "declarative facts") {
		t.Fatal("MemoryGuidance must contain 'declarative facts'")
	}
}

func TestMemoryGuidance_SwitchContract(t *testing.T) {
	assertGuidanceSwitch(t, guidanceSwitchCase{
		name:               "memory",
		guidance:           MemoryGuidance,
		build:              BuildMemoryGuidance,
		injectedEvidence:   "memory_guidance_injected",
		suppressedEvidence: "memory_guidance_suppressed",
	})
}

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

func TestModelPromptGuidanceFacade(t *testing.T) {
	if got := ModelPromptRole("gpt-5.1-codex"); got != PromptRoleDeveloper {
		t.Fatalf("ModelPromptRole facade=%q want %q", got, PromptRoleDeveloper)
	}
	result := BuildModelPromptGuidance(ModelPromptGuidanceOptions{
		Model:                  "gpt-5",
		ValidToolNames:         []string{"read_file"},
		ToolUseEnforcementMode: "always",
	})
	if result.PromptRole != PromptRoleDeveloper {
		t.Fatalf("PromptRole=%q want %q", result.PromptRole, PromptRoleDeveloper)
	}
	if result.Guidance == "" {
		t.Fatal("BuildModelPromptGuidance facade returned empty guidance")
	}
}

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
