package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/repair"

func SanitizeReasoningTags(text string) string {
	return repair.SanitizeReasoningTags(text)
}

func ContainsReasoningTagMarker(text string) bool {
	return repair.ContainsReasoningTagMarker(text)
}

func SanitizeReasoningStreamChunk(text string, inReasoning bool) (string, bool) {
	return repair.SanitizeReasoningStreamChunk(text, inReasoning)
}
