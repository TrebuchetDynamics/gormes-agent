package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/compression"

const (
	ManualCompressionSessionSplitCode     = compression.ManualCompressionSessionSplitCode
	ManualCompressionSessionUnchangedCode = compression.ManualCompressionSessionUnchangedCode
)

type ManualCompressionFeedback = compression.ManualCompressionFeedback
type ManualCompressionSessionEvidence = compression.ManualCompressionSessionEvidence

func SummarizeManualCompression(before, after []Message, beforeTokens, afterTokens int) ManualCompressionFeedback {
	return compression.SummarizeManualCompression(before, after, beforeTokens, afterTokens)
}

func ParseManualCompressionFocus(command string) string {
	return compression.ParseManualCompressionFocus(command)
}

func ManualCompressionSessionSplit(oldSessionID, newSessionID string) ManualCompressionSessionEvidence {
	return compression.ManualCompressionSessionSplit(oldSessionID, newSessionID)
}
