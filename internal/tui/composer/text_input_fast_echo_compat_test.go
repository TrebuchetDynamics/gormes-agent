package composer_test

import (
	"testing"

	tui "github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func TestTUITextInputFastEchoCompatibilityWrappers(t *testing.T) {
	if !tui.CanFastAppendShape("hello", 5, "!", 20, 5) {
		t.Fatal("CanFastAppendShape wrapper = false, want true")
	}
	if !tui.CanFastBackspaceShape("hello", 5, 20) {
		t.Fatal("CanFastBackspaceShape wrapper = false, want true")
	}
	if tui.SupportsFastEchoTerminal(map[string]string{"TERM_PROGRAM": "Apple_Terminal"}) {
		t.Fatal("SupportsFastEchoTerminal wrapper should reject Apple_Terminal")
	}
}
