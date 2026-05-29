package terminal

import (
	"strings"
	"testing"
)

var forbiddenCopyHotkeys = []string{
	"Cmd+C",
	"Ctrl+C",
	"Ctrl-Shift-C",
	"Cmd-Shift-C",
	"OSC 52",
	"clipboard hotkey",
	"Ink",
}

func TestNativeSelectionHelpExists(t *testing.T) {
	if NativeSelectionHelp == "" {
		t.Fatal("NativeSelectionHelp is empty")
	}
	if !strings.Contains(NativeSelectionHelp, "terminal") {
		t.Errorf("NativeSelectionHelp = %q; want it to contain substring %q", NativeSelectionHelp, "terminal")
	}
	if got := SelectionHelpLine(); got != NativeSelectionHelp {
		t.Errorf("SelectionHelpLine() = %q; want %q", got, NativeSelectionHelp)
	}
}

func TestNativeSelectionHelpNoFakeShortcuts(t *testing.T) {
	lower := strings.ToLower(NativeSelectionHelp)
	for _, bad := range forbiddenCopyHotkeys {
		if strings.Contains(lower, strings.ToLower(bad)) {
			t.Errorf("NativeSelectionHelp = %q contains forbidden shortcut %q", NativeSelectionHelp, bad)
		}
	}
}
