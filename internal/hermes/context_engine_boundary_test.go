package hermes

import (
	"context"
	"testing"
)

func TestContextEngineCompressionBoundaryVocabulary(t *testing.T) {
	engine := NewDisabledContextEngine("compression disabled by config")

	initial := engine.Status()
	if initial.Boundary.Status != ContextBoundaryStatusMissing {
		t.Fatalf("initial boundary status = %q, want %q", initial.Boundary.Status, ContextBoundaryStatusMissing)
	}
	if initial.Boundary.Last != nil {
		t.Fatalf("initial boundary last = %#v, want nil", initial.Boundary.Last)
	}

	boundary := CompressionBoundary{
		OldSessionID: "sess-before",
		NewSessionID: "sess-after",
		Reason:       "protected_head_tail_summary",
		CompressedAt: "2026-04-29T12:00:00Z",
	}
	if err := engine.OnCompressionBoundary(context.Background(), boundary); err != nil {
		t.Fatalf("OnCompressionBoundary: %v", err)
	}

	status := engine.Status()
	if status.Boundary.Status != ContextBoundaryStatusRecorded {
		t.Fatalf("boundary status = %q, want %q", status.Boundary.Status, ContextBoundaryStatusRecorded)
	}
	if status.Boundary.Last == nil {
		t.Fatal("boundary last = nil, want recorded boundary")
	}
	if *status.Boundary.Last != boundary {
		t.Fatalf("boundary last = %#v, want %#v", *status.Boundary.Last, boundary)
	}
	if status.CompressionCount != 1 {
		t.Fatalf("compression_count = %d, want 1", status.CompressionCount)
	}

	engine.OnSessionReset()
	reset := engine.Status()
	if reset.Boundary.Status != ContextBoundaryStatusMissing || reset.Boundary.Last != nil || reset.CompressionCount != 0 {
		t.Fatalf("reset status = %#v, want missing boundary and zero compression count", reset)
	}
}
