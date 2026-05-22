package hermes

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestLearningLoopSignalScorer_Deterministic(t *testing.T) {
	input := LearningLoopSignalInput{
		Transcript: []BackgroundReviewMessage{
			{Role: "user", Content: "please update the gateway docs"},
			{Role: "assistant", Content: "I'll patch the docs."},
		},
		NewUserTurns:        1,
		MemoryNudgeInterval: 2,
		SkillNudgeInterval:  10,
		ToolIterations:      10,
		EditPatches:         3,
	}

	first := ScoreLearningLoopSignals(input)
	second := ScoreLearningLoopSignals(input)

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("ScoreLearningLoopSignals() was not deterministic:\nfirst=%+v\nsecond=%+v", first, second)
	}
	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("json.Marshal(first): %v", err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("json.Marshal(second): %v", err)
	}
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("marshaled decisions differ:\nfirst=%s\nsecond=%s", firstJSON, secondJSON)
	}

	wantReasons := []string{"user_turn_threshold", "tool_iteration_threshold", "edit_burst"}
	if !reflect.DeepEqual(first.ReasonCodes, wantReasons) {
		t.Fatalf("ReasonCodes = %v, want %v", first.ReasonCodes, wantReasons)
	}
	if !first.ReviewMemory || !first.ReviewSkills {
		t.Fatalf("decision = %+v, want both memory and skills review", first)
	}
	if first.MemoryScore <= 0 || first.SkillScore <= 0 {
		t.Fatalf("scores = memory:%d skills:%d, want positive scores", first.MemoryScore, first.SkillScore)
	}
}

func TestLearningLoopSignalScorer_Thresholds(t *testing.T) {
	below := ScoreLearningLoopSignals(LearningLoopSignalInput{
		Transcript:          []BackgroundReviewMessage{{Role: "user", Content: "ping"}},
		NewUserTurns:        1,
		MemoryNudgeInterval: 10,
		SkillNudgeInterval:  10,
		ToolIterations:      1,
	})
	if below.ReviewMemory || below.ReviewSkills || len(below.ReasonCodes) != 0 {
		t.Fatalf("below-threshold decision = %+v, want no review", below)
	}

	hydrated := ScoreLearningLoopSignals(LearningLoopSignalInput{
		Transcript: []BackgroundReviewMessage{
			{Role: "user", Content: "q0"},
			{Role: "assistant", Content: "a0"},
			{Role: "user", Content: "q1"},
			{Role: "assistant", Content: "a1"},
			{Role: "user", Content: "q2"},
			{Role: "assistant", Content: "a2"},
			{Role: "user", Content: "q3"},
			{Role: "assistant", Content: "a3"},
			{Role: "user", Content: "q4"},
			{Role: "assistant", Content: "a4"},
			{Role: "user", Content: "q5"},
			{Role: "assistant", Content: "a5"},
			{Role: "user", Content: "q6"},
		},
		NewUserTurns:        3,
		MemoryNudgeInterval: 10,
		SkillNudgeInterval:  10,
	})
	if !hydrated.ReviewMemory {
		t.Fatalf("hydrated decision = %+v, want memory review", hydrated)
	}
	if got, want := hydrated.Evidence.TurnsSinceMemory, 10; got != want {
		t.Fatalf("TurnsSinceMemory = %d, want hydrated cadence at %d", got, want)
	}
	if !containsString(hydrated.ReasonCodes, "user_turn_threshold") {
		t.Fatalf("hydrated reasons = %v, want user_turn_threshold", hydrated.ReasonCodes)
	}

	skill := ScoreLearningLoopSignals(LearningLoopSignalInput{
		MemoryNudgeInterval: 10,
		SkillNudgeInterval:  10,
		ItersSinceSkill:     4,
		ToolIterations:      6,
	})
	if !skill.ReviewSkills || skill.ReviewMemory {
		t.Fatalf("skill threshold decision = %+v, want skills-only review", skill)
	}
	if !containsString(skill.ReasonCodes, "tool_iteration_threshold") {
		t.Fatalf("skill reasons = %v, want tool_iteration_threshold", skill.ReasonCodes)
	}
}

func TestLearningLoopSignalScorer_FeedbackAndRedaction(t *testing.T) {
	decision := ScoreLearningLoopSignals(LearningLoopSignalInput{
		Transcript: []BackgroundReviewMessage{
			{Role: "user", Content: "stop doing that; api key sk-live-secret should never appear"},
			{Role: "tool", Content: "patch failed with token ghp_secret"},
		},
		MemoryNudgeInterval: 10,
		SkillNudgeInterval:  10,
		RetryMarkers:        2,
		OperatorFeedback: []string{
			"don't format like this again",
			"remember this workflow correction",
		},
	})
	if !decision.ReviewSkills {
		t.Fatalf("decision = %+v, want skill review from operator feedback", decision)
	}
	for _, want := range []string{"retry_loop", "operator_feedback"} {
		if !containsString(decision.ReasonCodes, want) {
			t.Fatalf("reasons = %v, want %q", decision.ReasonCodes, want)
		}
	}
	if decision.Evidence.OperatorFeedbackCount != 2 || decision.Evidence.RedactedInputCount == 0 {
		t.Fatalf("evidence = %+v, want feedback count and redaction count", decision.Evidence)
	}
	raw, err := json.Marshal(decision)
	if err != nil {
		t.Fatalf("json.Marshal(decision): %v", err)
	}
	serialized := string(raw)
	for _, forbidden := range []string{"sk-live-secret", "ghp_secret", "stop doing that", "don't format like this", "remember this"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("decision leaked raw prompt/tool/feedback content %q: %s", forbidden, serialized)
		}
	}
}
