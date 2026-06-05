package reactions

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestOutcomeForFrame(t *testing.T) {
	tests := []struct {
		name      string
		phase     kernel.Phase
		cancelled bool
		want      ProcessingOutcome
	}{
		{name: "idle success", phase: kernel.PhaseIdle, want: ProcessingOutcomeSuccess},
		{name: "failed", phase: kernel.PhaseFailed, want: ProcessingOutcomeFailure},
		{name: "cancelling phase", phase: kernel.PhaseCancelling, want: ProcessingOutcomeCancelled},
		{name: "cancelled flag wins", phase: kernel.PhaseFailed, cancelled: true, want: ProcessingOutcomeCancelled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := OutcomeForFrame(tt.phase, tt.cancelled); got != tt.want {
				t.Fatalf("OutcomeForFrame(%q, %v) = %q, want %q", tt.phase, tt.cancelled, got, tt.want)
			}
		})
	}
}
