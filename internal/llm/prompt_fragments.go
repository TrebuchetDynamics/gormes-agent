package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts"

type PromptFragmentSource = prompts.PromptFragmentSource
type PromptFragmentRequest = prompts.PromptFragmentRequest
type PromptFragmentResult = prompts.PromptFragmentResult
type PromptFragmentEvidence = prompts.PromptFragmentEvidence
type PromptFragmentError = prompts.PromptFragmentError
type PromptFragmentCache = prompts.PromptFragmentCache

func NewPromptFragmentCache() *PromptFragmentCache {
	return prompts.NewPromptFragmentCache()
}

func RenderPromptFragment(req PromptFragmentRequest) (PromptFragmentResult, error) {
	return prompts.RenderPromptFragment(req)
}
