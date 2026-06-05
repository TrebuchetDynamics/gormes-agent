package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/guidance"

type PromptRole = guidance.PromptRole

const (
	PromptRoleSystem    PromptRole = guidance.PromptRoleSystem
	PromptRoleDeveloper PromptRole = guidance.PromptRoleDeveloper
)

type ModelPromptGuidanceOptions = guidance.ModelPromptGuidanceOptions
type ModelPromptGuidanceResult = guidance.ModelPromptGuidanceResult

func ModelPromptRole(model string) PromptRole {
	return guidance.ModelPromptRole(model)
}

func BuildModelPromptGuidance(opts ModelPromptGuidanceOptions) ModelPromptGuidanceResult {
	return guidance.BuildModelPromptGuidance(opts)
}
