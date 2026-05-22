package hermes

import "strings"

const (
	LearningLoopReasonUserTurnThreshold      = "user_turn_threshold"
	LearningLoopReasonToolIterationThreshold = "tool_iteration_threshold"
	LearningLoopReasonRetryLoop              = "retry_loop"
	LearningLoopReasonEditBurst              = "edit_burst"
	LearningLoopReasonOperatorFeedback       = "operator_feedback"
)

// LearningLoopSignalInput is the prompt-free feature vector for deciding
// whether a completed turn should request a background memory or skill review.
// Raw transcript/feedback text is accepted only so the scorer can count turns
// and redaction-worthy inputs; emitted evidence never includes the text itself.
type LearningLoopSignalInput struct {
	Transcript []BackgroundReviewMessage

	NewUserTurns int

	PriorUserTurnCount  int
	TurnsSinceMemory    int
	MemoryNudgeInterval int

	ItersSinceSkill    int
	ToolIterations     int
	SkillNudgeInterval int

	RetryMarkers int
	EditPatches  int

	OperatorFeedback []string
}

// LearningLoopSignalDecision is the deterministic, redacted decision object
// that can later gate a background review fork without invoking a provider or
// mutating memory/skills by itself.
type LearningLoopSignalDecision struct {
	ReviewMemory bool                       `json:"review_memory"`
	ReviewSkills bool                       `json:"review_skills"`
	MemoryScore  int                        `json:"memory_score"`
	SkillScore   int                        `json:"skill_score"`
	ReasonCodes  []string                   `json:"reason_codes"`
	Evidence     LearningLoopSignalEvidence `json:"evidence"`
}

// LearningLoopSignalEvidence contains counts only. It is safe to log because it
// excludes raw prompt text, tool output, secrets, and full transcripts.
type LearningLoopSignalEvidence struct {
	TranscriptMessages    int `json:"transcript_messages"`
	PriorUserTurns        int `json:"prior_user_turns"`
	NewUserTurns          int `json:"new_user_turns"`
	TurnsSinceMemory      int `json:"turns_since_memory"`
	ToolIterations        int `json:"tool_iterations"`
	ItersSinceSkill       int `json:"iters_since_skill"`
	RetryMarkers          int `json:"retry_markers"`
	EditPatches           int `json:"edit_patches"`
	OperatorFeedbackCount int `json:"operator_feedback_count"`
	RedactedInputCount    int `json:"redacted_input_count"`
}

// ScoreLearningLoopSignals derives explainable memory/skill review requests
// from counters and redacted transcript features. It is deliberately pure:
// callers own scheduling, provider calls, memory writes, and skill mutation.
func ScoreLearningLoopSignals(input LearningLoopSignalInput) LearningLoopSignalDecision {
	transcriptUserTurns := countTranscriptUserTurns(input.Transcript)
	priorUserTurns := nonNegative(input.PriorUserTurnCount)
	if priorUserTurns == 0 {
		priorUserTurns = transcriptUserTurns
	}

	newUserTurns := nonNegative(input.NewUserTurns)
	turnsSinceMemory := nonNegative(input.TurnsSinceMemory)
	if turnsSinceMemory == 0 && input.MemoryNudgeInterval > 0 && priorUserTurns > 0 {
		turnsSinceMemory = priorUserTurns % input.MemoryNudgeInterval
	}
	turnsSinceMemory += newUserTurns

	itersSinceSkill := nonNegative(input.ItersSinceSkill) + nonNegative(input.ToolIterations)
	retryMarkers := nonNegative(input.RetryMarkers)
	editPatches := nonNegative(input.EditPatches)
	feedbackCount := len(input.OperatorFeedback)

	decision := LearningLoopSignalDecision{
		Evidence: LearningLoopSignalEvidence{
			TranscriptMessages:    len(input.Transcript),
			PriorUserTurns:        priorUserTurns,
			NewUserTurns:          newUserTurns,
			TurnsSinceMemory:      turnsSinceMemory,
			ToolIterations:        nonNegative(input.ToolIterations),
			ItersSinceSkill:       itersSinceSkill,
			RetryMarkers:          retryMarkers,
			EditPatches:           editPatches,
			OperatorFeedbackCount: feedbackCount,
			RedactedInputCount:    countRedactedSignalInputs(input),
		},
	}

	if input.MemoryNudgeInterval > 0 && turnsSinceMemory >= input.MemoryNudgeInterval {
		decision.ReviewMemory = true
		decision.MemoryScore += 100
		decision.ReasonCodes = append(decision.ReasonCodes, LearningLoopReasonUserTurnThreshold)
	}
	if input.SkillNudgeInterval > 0 && itersSinceSkill >= input.SkillNudgeInterval {
		decision.ReviewSkills = true
		decision.SkillScore += 100
		decision.ReasonCodes = append(decision.ReasonCodes, LearningLoopReasonToolIterationThreshold)
	}
	if retryMarkers >= 2 {
		decision.ReviewSkills = true
		decision.SkillScore += 40
		decision.ReasonCodes = append(decision.ReasonCodes, LearningLoopReasonRetryLoop)
	}
	if editPatches >= 3 {
		decision.ReviewSkills = true
		decision.SkillScore += 30
		decision.ReasonCodes = append(decision.ReasonCodes, LearningLoopReasonEditBurst)
	}
	if feedbackCount > 0 {
		decision.ReviewSkills = true
		decision.SkillScore += 50
		decision.ReasonCodes = append(decision.ReasonCodes, LearningLoopReasonOperatorFeedback)
	}

	return decision
}

func countTranscriptUserTurns(messages []BackgroundReviewMessage) int {
	var count int
	for _, msg := range messages {
		if strings.EqualFold(strings.TrimSpace(msg.Role), "user") {
			count++
		}
	}
	return count
}

func countRedactedSignalInputs(input LearningLoopSignalInput) int {
	var count int
	for _, msg := range input.Transcript {
		if looksSecretish(msg.Content) {
			count++
		}
	}
	for _, feedback := range input.OperatorFeedback {
		if strings.TrimSpace(feedback) != "" {
			count++
		}
	}
	return count
}

func looksSecretish(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{"api key", "token", "secret", "sk-", "ghp_"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func nonNegative(value int) int {
	if value < 0 {
		return 0
	}
	return value
}
