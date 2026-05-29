package agent

import (
	"fmt"
	"testing"
)

func TestLoopDetector_HardLoop(t *testing.T) {
	d := NewLoopDetector()
	for i := 0; i < 3; i++ {
		d.Record(TurnRecord{Index: i, ToolCalls: []string{"read_file"}, Response: "ok"})
	}
	result := d.Check()
	if !result.Detected {
		t.Fatal("expected hard loop detection")
	}
	if result.Type != LoopTypeHardLoop {
		t.Fatalf("expected hard_loop, got %s", result.Type)
	}
}

func TestLoopDetector_FailingLoop(t *testing.T) {
	d := NewLoopDetector()
	for i := 0; i < 3; i++ {
		d.Record(TurnRecord{Index: i, HadError: true, Response: fmt.Sprintf("error%d", i), ToolCalls: []string{"cmd"}})
	}
	result := d.Check()
	if !result.Detected {
		t.Fatal("expected failing loop detection")
	}
	if result.Type != LoopTypeFailingLoop {
		t.Fatalf("expected failing_loop, got %s", result.Type)
	}
}

func TestLoopDetector_TextRepetition(t *testing.T) {
	d := NewLoopDetector()
	for i := 0; i < 3; i++ {
		d.Record(TurnRecord{Index: i, Response: "same response", ToolCalls: []string{fmt.Sprintf("cmd%d", i)}})
	}
	result := d.Check()
	if !result.Detected {
		t.Fatal("expected text repetition detection")
	}
	if result.Type != LoopTypeTextRepetition {
		t.Fatalf("expected text_repetition, got %s", result.Type)
	}
}

func TestLoopDetector_NoAction(t *testing.T) {
	d := NewLoopDetector()
	for i := 0; i < 3; i++ {
		d.Record(TurnRecord{Index: i, ToolCalls: []string{}, Response: fmt.Sprintf("waiting%d", i)})
	}
	result := d.Check()
	if !result.Detected {
		t.Fatal("expected no-action detection")
	}
	if result.Type != LoopTypeNoAction {
		t.Fatalf("expected no_action, got %s", result.Type)
	}
}

func TestLoopDetector_SameTool(t *testing.T) {
	d := NewLoopDetector()
	for i := 0; i < 3; i++ {
		d.Record(TurnRecord{Index: i, ToolCalls: []string{"read_file", "write_file"}, Response: fmt.Sprintf("step%d", i)})
	}
	result := d.Check()
	if !result.Detected {
		t.Fatal("expected same-tool detection")
	}
	if result.Type != LoopTypeSameTool {
		t.Fatalf("expected same_tool, got %s", result.Type)
	}
}

func TestLoopDetector_NoLoop(t *testing.T) {
	d := NewLoopDetector()
	d.Record(TurnRecord{Index: 0, ToolCalls: []string{"read_file"}})
	d.Record(TurnRecord{Index: 1, ToolCalls: []string{"write_file"}})
	result := d.Check()
	if result.Detected {
		t.Fatal("expected no loop detection")
	}
}

func TestLoopDetector_ConfigurableThresholds(t *testing.T) {
	d := NewLoopDetector()
	d.HardLoopThreshold = 5
	d.SameToolThreshold = 5
	d.TextRepetitionThreshold = 5
	for i := 0; i < 4; i++ {
		d.Record(TurnRecord{Index: i, ToolCalls: []string{"read_file"}, Response: "ok"})
	}
	result := d.Check()
	if result.Detected {
		t.Fatalf("expected no loop below threshold, got %s", result.Type)
	}
	d.Record(TurnRecord{Index: 4, ToolCalls: []string{"read_file"}, Response: "ok"})
	result = d.Check()
	if !result.Detected {
		t.Fatal("expected loop at threshold")
	}
}

func TestLoopDetector_EvidenceContainsType(t *testing.T) {
	d := NewLoopDetector()
	for i := 0; i < 3; i++ {
		d.Record(TurnRecord{Index: i, HadError: true, Response: fmt.Sprintf("err%d", i)})
	}
	result := d.Check()
	if result.Evidence == "" {
		t.Fatal("expected non-empty evidence")
	}
}
