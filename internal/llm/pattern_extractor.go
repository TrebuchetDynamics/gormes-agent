package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/learning"

type ToolSequenceObservation = learning.ToolSequenceObservation
type ReasoningPattern = learning.ReasoningPattern
type ToolSequence = learning.ToolSequence
type BehavioralKnowledge = learning.BehavioralKnowledge
type PatternExtractor = learning.PatternExtractor

func NewPatternExtractor() *PatternExtractor {
	return learning.NewPatternExtractor()
}
