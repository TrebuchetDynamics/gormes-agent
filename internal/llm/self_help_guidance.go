package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/guidance"

func GormesSelfHelpGuidanceForPrompt(userPrompt string) (string, bool) {
	return guidance.GormesSelfHelpGuidanceForPrompt(userPrompt)
}
