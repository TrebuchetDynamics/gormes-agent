package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts"

type PromptAssemblyOptions = prompts.PromptAssemblyOptions
type PromptAssemblyResult = prompts.PromptAssemblyResult
type PromptBlockEvidence = prompts.PromptBlockEvidence

func BuildSystemPrompt(opts PromptAssemblyOptions) PromptAssemblyResult {
	return prompts.BuildSystemPrompt(opts)
}
