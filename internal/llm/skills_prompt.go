package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts"

type SkillsPromptOptions = prompts.SkillsPromptOptions
type SkillsPromptEvidence = prompts.SkillsPromptEvidence

func ResetSkillsPromptCacheForTest() {
	prompts.ResetSkillsPromptCacheForTest()
}

func BuildSkillsSystemPrompt(opts SkillsPromptOptions) (string, []SkillsPromptEvidence, error) {
	return prompts.BuildSkillsSystemPrompt(opts)
}
