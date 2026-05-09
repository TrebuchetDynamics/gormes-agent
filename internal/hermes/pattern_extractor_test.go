package hermes

import (
	"strings"
	"testing"
)

func TestPatternExtractor_SuccessfulPatterns(t *testing.T) {
	pe := NewPatternExtractor()
	for i := 0; i < 5; i++ {
		pe.RecordSequence([]string{"read", "edit", "write"}, true)
	}
	pe.RecordSequence([]string{"read", "edit", "write"}, false)
	for i := 0; i < 4; i++ {
		pe.RecordSequence([]string{"read", "write"}, true)
	}
	for i := 0; i < 2; i++ {
		pe.RecordSequence([]string{"read", "write"}, false)
	}

	success := pe.SuccessfulPatterns(5)
	if len(success) != 1 {
		t.Fatalf("got %d successful patterns, want 1", len(success))
	}
	got := success[0]
	if got.Count != 5 || got.Successes != 5 || got.Failures != 1 || got.Observations != 6 {
		t.Fatalf("pattern counts = count:%d successes:%d failures:%d observations:%d, want 5/5/1/6", got.Count, got.Successes, got.Failures, got.Observations)
	}
	if got.SuccessRate <= 0.80 {
		t.Fatalf("success rate = %.3f, want > 0.80", got.SuccessRate)
	}
}

func TestPatternExtractor_FailedPatterns(t *testing.T) {
	pe := NewPatternExtractor()
	for i := 0; i < 5; i++ {
		pe.RecordSequence([]string{"read", "rm"}, false)
	}
	pe.RecordSequence([]string{"read", "rm"}, true)
	for i := 0; i < 2; i++ {
		pe.RecordSequence([]string{"read", "write"}, false)
		pe.RecordSequence([]string{"read", "write"}, true)
	}

	failed := pe.FailedPatterns(5)
	if len(failed) != 1 {
		t.Fatalf("got %d failed patterns, want 1", len(failed))
	}
	got := failed[0]
	if got.Count != 5 || got.Successes != 1 || got.Failures != 5 || got.Observations != 6 {
		t.Fatalf("pattern counts = count:%d successes:%d failures:%d observations:%d, want 5/1/5/6", got.Count, got.Successes, got.Failures, got.Observations)
	}
	if got.SuccessRate >= 0.30 {
		t.Fatalf("success rate = %.3f, want < 0.30", got.SuccessRate)
	}
}

func TestPatternExtractor_ReasoningPatternsFromSuccessfulObservations(t *testing.T) {
	pe := NewPatternExtractor()
	for i := 0; i < 5; i++ {
		pe.RecordObservation(ToolSequenceObservation{
			SessionID:        "session-good",
			Tools:            []string{"search_files", "read_file", "patch"},
			Success:          true,
			ReasoningPattern: "inspect the local context before editing",
		})
	}
	pe.RecordObservation(ToolSequenceObservation{
		SessionID:        "session-bad",
		Tools:            []string{"search_files", "read_file", "patch"},
		Success:          false,
		ReasoningPattern: "guess without reading",
	})

	success := pe.SuccessfulPatterns(5)
	if len(success) != 1 {
		t.Fatalf("got %d successful patterns, want 1", len(success))
	}
	reasoning := success[0].ReasoningPatterns
	if len(reasoning) != 1 {
		t.Fatalf("got %d reasoning patterns, want 1: %#v", len(reasoning), reasoning)
	}
	if reasoning[0].Text != "inspect the local context before editing" || reasoning[0].Count != 5 {
		t.Fatalf("reasoning pattern = %#v, want successful reasoning counted only", reasoning[0])
	}

	summary := pe.PatternSummary()
	if !strings.Contains(summary, "inspect the local context before editing (5)") {
		t.Fatalf("summary missing reasoning evidence:\n%s", summary)
	}
}

func TestPatternExtractor_GonchoKnowledgeRecordsAreStructured(t *testing.T) {
	pe := NewPatternExtractor()
	for i := 0; i < 5; i++ {
		pe.RecordObservation(ToolSequenceObservation{
			SessionID:        "session-success",
			Tools:            []string{"read_file", "patch"},
			Success:          true,
			ReasoningPattern: "read then patch",
		})
		pe.RecordSequence([]string{"read_file", "terminal"}, false)
	}
	pe.RecordSequence([]string{"read_file", "patch"}, false)

	knowledge := pe.GonchoKnowledge(5)
	if len(knowledge) != 2 {
		t.Fatalf("got %d knowledge records, want 2: %#v", len(knowledge), knowledge)
	}

	success := findKnowledge(t, knowledge, "tool_sequence_success_pattern")
	if success.Source != "pattern_extractor" {
		t.Fatalf("success source = %q, want pattern_extractor", success.Source)
	}
	if success.Observations != 6 || success.Successes != 5 || success.Failures != 1 || success.SuccessRate <= 0.80 {
		t.Fatalf("success knowledge = %#v, want 5/1 with rate > 0.80", success)
	}
	if !patternContainsString(success.Tags, "behavioral_pattern") || !patternContainsString(success.Tags, "tool_sequence") {
		t.Fatalf("success tags = %#v, want behavioral_pattern and tool_sequence", success.Tags)
	}
	if !strings.Contains(success.Content, "read_file -> patch") || !strings.Contains(success.Content, "read then patch") {
		t.Fatalf("success content missing sequence or reasoning evidence: %q", success.Content)
	}

	anti := findKnowledge(t, knowledge, "tool_sequence_anti_pattern")
	if anti.Observations != 5 || anti.Failures != 5 || anti.SuccessRate != 0 {
		t.Fatalf("anti-pattern knowledge = %#v, want all failed observations", anti)
	}
	if !patternContainsString(anti.Tags, "anti_pattern") {
		t.Fatalf("anti-pattern tags = %#v, want anti_pattern", anti.Tags)
	}
}

func TestPatternExtractor_Empty(t *testing.T) {
	pe := NewPatternExtractor()
	if len(pe.SuccessfulPatterns(1)) != 0 {
		t.Fatal("empty extractor should have no patterns")
	}
}

func findKnowledge(t *testing.T, records []BehavioralKnowledge, kind string) BehavioralKnowledge {
	t.Helper()
	for _, record := range records {
		if record.Kind == kind {
			return record
		}
	}
	t.Fatalf("missing knowledge kind %q in %#v", kind, records)
	return BehavioralKnowledge{}
}

func patternContainsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
