package repair

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/repair/reasoning"

// SanitizeReasoningTags returns visible assistant text with inline reasoning
// XML blocks removed. Callers must keep raw stream/transcript text separately
// for audit rather than treating this sanitized copy as source evidence.
func SanitizeReasoningTags(text string) string {
	return reasoning.SanitizeTags(text)
}

// ContainsReasoningTagMarker reports whether text contains a provider-leaked
// reasoning XML marker. It is a cheap guard for the streaming sanitizer.
func ContainsReasoningTagMarker(text string) bool {
	return reasoning.ContainsTagMarker(text)
}

// SanitizeReasoningStreamChunk removes provider-leaked reasoning XML blocks
// from one streamed content chunk while preserving state across chunks. The
// returned bool is true when the next chunk starts inside a reasoning block.
func SanitizeReasoningStreamChunk(text string, inReasoning bool) (string, bool) {
	return reasoning.SanitizeStreamChunk(text, inReasoning)
}
