package tui

import "testing"

func TestTUITextInputFastEchoCompatibilityWrappers(t *testing.T) {
	if !CanFastAppendShape("hello", 5, "!", 20, 5) {
		t.Fatal("CanFastAppendShape wrapper = false, want true")
	}
	if !CanFastBackspaceShape("hello", 5, 20) {
		t.Fatal("CanFastBackspaceShape wrapper = false, want true")
	}
	if SupportsFastEchoTerminal(map[string]string{"TERM_PROGRAM": "Apple_Terminal"}) {
		t.Fatal("SupportsFastEchoTerminal wrapper should reject Apple_Terminal")
	}
}
