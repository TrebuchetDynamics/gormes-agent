package guidance

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/guidance/selfhelp"

// GormesSelfHelpGuidanceForPrompt returns the deterministic self-help prompt
// block only for user prompts that ask about operating Gormes itself.
func GormesSelfHelpGuidanceForPrompt(userPrompt string) (string, bool) {
	return selfhelp.GuidanceForPrompt(userPrompt)
}
