package prompts

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/skillprompt"

type SkillsPromptOptions = skillprompt.SkillsPromptOptions
type SkillsPromptEvidence = skillprompt.SkillsPromptEvidence

func ResetSkillsPromptCacheForTest() {
	skillprompt.ResetSkillsPromptCacheForTest()
}

func BuildSkillsSystemPrompt(opts SkillsPromptOptions) (string, []SkillsPromptEvidence, error) {
	return skillprompt.BuildSkillsSystemPrompt(opts)
}
