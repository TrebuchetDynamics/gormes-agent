package agent

import "fmt"

type LoopType string

const (
	LoopTypeHardLoop       LoopType = "hard_loop"
	LoopTypeFailingLoop    LoopType = "failing_loop"
	LoopTypeTextRepetition LoopType = "text_repetition"
	LoopTypeNoAction       LoopType = "no_action"
	LoopTypeSameTool       LoopType = "same_tool"
)

type LoopDetector struct {
	HardLoopThreshold       int
	FailingLoopThreshold    int
	TextRepetitionThreshold int
	NoActionThreshold       int
	SameToolThreshold       int

	history []TurnRecord
}

type TurnRecord struct {
	Index     int
	ToolCalls []string
	Response  string
	HadError  bool
}

type LoopDetectionResult struct {
	Detected bool
	Type     LoopType
	Evidence string
}

func NewLoopDetector() *LoopDetector {
	return &LoopDetector{
		HardLoopThreshold:       3,
		FailingLoopThreshold:    3,
		TextRepetitionThreshold: 2,
		NoActionThreshold:       3,
		SameToolThreshold:       3,
	}
}

func (d *LoopDetector) Record(turn TurnRecord) {
	d.history = append(d.history, turn)
}

func (d *LoopDetector) Check() LoopDetectionResult {
	if result := d.checkHardLoop(); result.Detected {
		return result
	}
	if result := d.checkFailingLoop(); result.Detected {
		return result
	}
	if result := d.checkTextRepetition(); result.Detected {
		return result
	}
	if result := d.checkNoAction(); result.Detected {
		return result
	}
	if result := d.checkSameTool(); result.Detected {
		return result
	}
	return LoopDetectionResult{Detected: false}
}

func (d *LoopDetector) checkHardLoop() LoopDetectionResult {
	if len(d.history) < d.HardLoopThreshold {
		return LoopDetectionResult{Detected: false}
	}
	last := d.history[len(d.history)-1]
	count := countTrailingTurns(d.history, func(turn TurnRecord) bool {
		return turnsEqual(turn, last)
	})
	if count >= d.HardLoopThreshold {
		return LoopDetectionResult{
			Detected: true,
			Type:     LoopTypeHardLoop,
			Evidence: fmt.Sprintf("hard_loop_detected: %d identical turns", count),
		}
	}
	return LoopDetectionResult{Detected: false}
}

func (d *LoopDetector) checkFailingLoop() LoopDetectionResult {
	if len(d.history) < d.FailingLoopThreshold {
		return LoopDetectionResult{Detected: false}
	}
	count := countTrailingTurns(d.history, func(turn TurnRecord) bool {
		return turn.HadError
	})
	if count >= d.FailingLoopThreshold {
		return LoopDetectionResult{
			Detected: true,
			Type:     LoopTypeFailingLoop,
			Evidence: fmt.Sprintf("failing_loop_detected: %d consecutive errors", count),
		}
	}
	return LoopDetectionResult{Detected: false}
}

func (d *LoopDetector) checkTextRepetition() LoopDetectionResult {
	if len(d.history) < d.TextRepetitionThreshold+1 {
		return LoopDetectionResult{Detected: false}
	}
	last := d.history[len(d.history)-1].Response
	count := 0
	for i := len(d.history) - 2; i >= 0; i-- {
		if d.history[i].Response == last {
			count++
		} else {
			break
		}
	}
	if count >= d.TextRepetitionThreshold {
		return LoopDetectionResult{
			Detected: true,
			Type:     LoopTypeTextRepetition,
			Evidence: fmt.Sprintf("text_repetition_detected: %d identical responses", count+1),
		}
	}
	return LoopDetectionResult{Detected: false}
}

func (d *LoopDetector) checkNoAction() LoopDetectionResult {
	if len(d.history) < d.NoActionThreshold {
		return LoopDetectionResult{Detected: false}
	}
	count := countTrailingTurns(d.history, func(turn TurnRecord) bool {
		return len(turn.ToolCalls) == 0
	})
	if count >= d.NoActionThreshold {
		return LoopDetectionResult{
			Detected: true,
			Type:     LoopTypeNoAction,
			Evidence: fmt.Sprintf("no_action_detected: %d turns without tool calls", count),
		}
	}
	return LoopDetectionResult{Detected: false}
}

func (d *LoopDetector) checkSameTool() LoopDetectionResult {
	if len(d.history) < d.SameToolThreshold {
		return LoopDetectionResult{Detected: false}
	}
	lastTools := d.history[len(d.history)-1].ToolCalls
	if len(lastTools) == 0 {
		return LoopDetectionResult{Detected: false}
	}
	count := countTrailingTurns(d.history, func(turn TurnRecord) bool {
		return slicesEqual(turn.ToolCalls, lastTools)
	})
	if count >= d.SameToolThreshold {
		return LoopDetectionResult{
			Detected: true,
			Type:     LoopTypeSameTool,
			Evidence: fmt.Sprintf("same_tool_detected: %d consecutive turns with identical tool calls (%v)", count, lastTools),
		}
	}
	return LoopDetectionResult{Detected: false}
}

func countTrailingTurns(history []TurnRecord, match func(TurnRecord) bool) int {
	count := 0
	for i := len(history) - 1; i >= 0; i-- {
		if match(history[i]) {
			count++
		} else {
			break
		}
	}
	return count
}

func turnsEqual(a, b TurnRecord) bool {
	if a.HadError != b.HadError {
		return false
	}
	if a.Response != b.Response {
		return false
	}
	return slicesEqual(a.ToolCalls, b.ToolCalls)
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
