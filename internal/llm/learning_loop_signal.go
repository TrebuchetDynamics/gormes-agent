package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/learning"

const (
	LearningLoopReasonUserTurnThreshold      = learning.LearningLoopReasonUserTurnThreshold
	LearningLoopReasonToolIterationThreshold = learning.LearningLoopReasonToolIterationThreshold
	LearningLoopReasonRetryLoop              = learning.LearningLoopReasonRetryLoop
	LearningLoopReasonEditBurst              = learning.LearningLoopReasonEditBurst
	LearningLoopReasonOperatorFeedback       = learning.LearningLoopReasonOperatorFeedback
)

type LearningLoopSignalInput = learning.LearningLoopSignalInput
type LearningLoopSignalDecision = learning.LearningLoopSignalDecision
type LearningLoopSignalEvidence = learning.LearningLoopSignalEvidence

func ScoreLearningLoopSignals(input LearningLoopSignalInput) LearningLoopSignalDecision {
	return learning.ScoreLearningLoopSignals(input)
}
