package keybindings

import "testing"

func TestStateClassifiesEquivalentConflictAndMissing(t *testing.T) {
	desired := terminalSendBinding("cmd+c", "terminalFocus && terminalTextSelected", "\x1b[99;13u")

	if state, conflictKey := State([]map[string]any{desired}, desired); state != Equivalent || conflictKey != "" {
		t.Fatalf("State(equivalent) = %v, %q; want Equivalent, empty conflict", state, conflictKey)
	}

	conflicting := []map[string]any{{"key": "cmd+c", "command": "workbench.action.terminal.copySelection", "when": "terminalFocus"}}
	if state, conflictKey := State(conflicting, desired); state != Conflict || conflictKey != "cmd+c" {
		t.Fatalf("State(conflict) = %v, %q; want Conflict, cmd+c", state, conflictKey)
	}

	disjoint := []map[string]any{terminalSendBinding("cmd+c", "terminalFocus && !terminalTextSelected", "\x03")}
	if state, conflictKey := State(disjoint, desired); state != Missing || conflictKey != "" {
		t.Fatalf("State(disjoint) = %v, %q; want Missing, empty conflict", state, conflictKey)
	}
}

func TestAnalyzeTerminalKeybindingsReportsMissingCompleteAndConflict(t *testing.T) {
	missing := AnalyzeTerminalKeybindings(nil, "linux")
	if missing.Complete() || len(missing.Missing) == 0 || missing.ConflictKey != "" {
		t.Fatalf("AnalyzeTerminalKeybindings(missing) = %+v, want missing entries and no conflict", missing)
	}

	complete := AnalyzeTerminalKeybindings(DefaultTerminalKeybindings("linux"), "linux")
	if !complete.Complete() {
		t.Fatalf("AnalyzeTerminalKeybindings(complete) = %+v, want complete", complete)
	}

	conflict := AnalyzeTerminalKeybindings([]map[string]any{{"key": "cmd+z", "command": "workbench.action.terminal.undo", "when": "terminalFocus"}}, "linux")
	if conflict.ConflictKey != "cmd+z" {
		t.Fatalf("AnalyzeTerminalKeybindings(conflict) = %+v, want cmd+z conflict", conflict)
	}
}

func TestDefaultTerminalKeybindingsAddsDarwinCopyBindingOnly(t *testing.T) {
	if !hasKey(DefaultTerminalKeybindings("darwin"), "cmd+c") {
		t.Fatal("DefaultTerminalKeybindings(darwin) missing cmd+c")
	}
	if hasKey(DefaultTerminalKeybindings("linux"), "cmd+c") {
		t.Fatal("DefaultTerminalKeybindings(linux) unexpectedly includes cmd+c")
	}
}

func hasKey(bindings []map[string]any, key string) bool {
	for _, binding := range bindings {
		if stringField(binding, "key") == key {
			return true
		}
	}
	return false
}
