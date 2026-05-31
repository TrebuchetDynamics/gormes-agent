package prompts

import promptassembly "github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/assembly"

type PromptAssemblyOptions = promptassembly.PromptAssemblyOptions
type PromptAssemblyResult = promptassembly.PromptAssemblyResult
type PromptBlockEvidence = promptassembly.PromptBlockEvidence

func BuildSystemPrompt(opts PromptAssemblyOptions) PromptAssemblyResult {
	return promptassembly.BuildSystemPrompt(opts)
}
