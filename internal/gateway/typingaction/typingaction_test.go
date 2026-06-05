package typingaction

import (
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestShouldSendForPhase(t *testing.T) {
	for _, phase := range []kernel.Phase{kernel.PhaseConnecting, kernel.PhaseStreaming, kernel.PhaseReconnecting, kernel.PhaseFinalizing} {
		if !ShouldSendForPhase(phase) {
			t.Fatalf("ShouldSendForPhase(%v) = false, want true", phase)
		}
	}
	for _, phase := range []kernel.Phase{kernel.PhaseIdle, kernel.PhaseFailed, kernel.PhaseCancelling} {
		if ShouldSendForPhase(phase) {
			t.Fatalf("ShouldSendForPhase(%v) = true, want false", phase)
		}
	}
}

func TestConstants(t *testing.T) {
	if Name != "typing" || FailedCode != "typing_action_failed" || ThrottleWindow != 4*time.Second {
		t.Fatalf("constants = %q %q %s", Name, FailedCode, ThrottleWindow)
	}
}
