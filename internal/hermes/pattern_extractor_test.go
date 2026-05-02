package hermes

import (
	"testing"
)

func TestPatternExtractor_SuccessfulPatterns(t *testing.T) {
	pe := NewPatternExtractor()
	pe.RecordSequence([]string{"read", "edit", "write"}, true)
	pe.RecordSequence([]string{"read", "edit", "write"}, true)
	pe.RecordSequence([]string{"read", "write"}, false)

	success := pe.SuccessfulPatterns(2)
	if len(success) != 1 {
		t.Fatalf("got %d successful patterns, want 1", len(success))
	}
	if success[0].Count != 2 {
		t.Fatalf("pattern count = %d, want 2", success[0].Count)
	}
}

func TestPatternExtractor_FailedPatterns(t *testing.T) {
	pe := NewPatternExtractor()
	pe.RecordSequence([]string{"read", "rm"}, false)
	pe.RecordSequence([]string{"read", "rm"}, false)
	pe.RecordSequence([]string{"read", "rm"}, false)
	pe.RecordSequence([]string{"read", "write"}, true)

	failed := pe.FailedPatterns(3)
	if len(failed) != 1 {
		t.Fatalf("got %d failed patterns, want 1", len(failed))
	}
}

func TestPatternExtractor_Summary(t *testing.T) {
	pe := NewPatternExtractor()
	pe.RecordSequence([]string{"read", "write"}, true)
	pe.RecordSequence([]string{"read", "rm"}, false)

	summary := pe.PatternSummary()
	if summary == "" {
		t.Fatal("summary should not be empty")
	}
}

func TestPatternExtractor_Empty(t *testing.T) {
	pe := NewPatternExtractor()
	if len(pe.SuccessfulPatterns(1)) != 0 {
		t.Fatal("empty extractor should have no patterns")
	}
}
