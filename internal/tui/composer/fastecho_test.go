package composer

import "testing"

func TestFastEchoShape(t *testing.T) {
	if !CanFastAppendShape("hello", 5, "!", 20, 5) {
		t.Fatal("ascii append at cursor should be fast-echo safe")
	}
	if CanFastAppendShape("hello", 3, "!", 20, 5) {
		t.Fatal("append away from cursor should not be fast-echo safe")
	}
	if CanFastAppendShape("hello", 5, "界", 20, 5) {
		t.Fatal("non-ascii append should not be fast-echo safe")
	}
	if !CanFastBackspaceShape("hello", 5, 20) {
		t.Fatal("ascii backspace at cursor should be fast-echo safe")
	}
	if CanFastBackspaceShape("hello ", 6, 6) {
		t.Fatal("backspace at wrap boundary should not be fast-echo safe")
	}
}

func TestSupportsFastEchoTerminal(t *testing.T) {
	if SupportsFastEchoTerminal(map[string]string{"TERM_PROGRAM": "Apple_Terminal"}) {
		t.Fatal("Apple Terminal should disable fast echo")
	}
	if !SupportsFastEchoTerminal(nil) {
		t.Fatal("nil env should allow fast echo")
	}
}
