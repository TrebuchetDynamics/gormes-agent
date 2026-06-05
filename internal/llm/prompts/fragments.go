package prompts

import promptfragments "github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/fragments"

type PromptFragmentSource = promptfragments.PromptFragmentSource
type PromptFragmentRequest = promptfragments.PromptFragmentRequest
type PromptFragmentResult = promptfragments.PromptFragmentResult
type PromptFragmentEvidence = promptfragments.PromptFragmentEvidence
type PromptFragmentError = promptfragments.PromptFragmentError
type PromptFragmentCache = promptfragments.PromptFragmentCache

func NewPromptFragmentCache() *PromptFragmentCache {
	return promptfragments.NewPromptFragmentCache()
}

func RenderPromptFragment(req PromptFragmentRequest) (PromptFragmentResult, error) {
	return promptfragments.RenderPromptFragment(req)
}
