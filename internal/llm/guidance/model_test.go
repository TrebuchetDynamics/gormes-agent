package guidance

import "testing"

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
