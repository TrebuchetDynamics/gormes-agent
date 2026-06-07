// internal/core/subagent/blocked_test.go
package lifecycle

import "testing"

func TestLifecycleConstants(t *testing.T) {
	if MaxDepth != 2 {
		t.Errorf("MaxDepth: want 2, got %d", MaxDepth)
	}
	if DefaultMaxConcurrent != 3 {
		t.Errorf("DefaultMaxConcurrent: want 3, got %d", DefaultMaxConcurrent)
	}
	if DefaultMaxIterations != 50 {
		t.Errorf("DefaultMaxIterations: want 50, got %d", DefaultMaxIterations)
	}
}

func TestBlockedToolRequestTrimsAndSkipsEmptyNames(t *testing.T) {
	got := BlockedToolRequest([]string{" ", " echo ", " delegate_task "})
	if got != "delegate_task" {
		t.Fatalf("BlockedToolRequest = %q, want delegate_task", got)
	}
}

func TestToolAllowlistedTrimsNamesAndRejectsBlockedTools(t *testing.T) {
	if !ToolAllowlisted([]string{" echo "}, " echo ") {
		t.Fatal("ToolAllowlisted trimmed echo = false, want true")
	}
	if ToolAllowlisted([]string{" delegate_task "}, " delegate_task ") {
		t.Fatal("ToolAllowlisted delegate_task = true, want false for blocked tool")
	}
	if ToolAllowlisted(nil, " memory ") {
		t.Fatal("ToolAllowlisted default memory = true, want false for blocked tool")
	}
}

func TestBlockedToolsForwardLooking(t *testing.T) {
	want := []string{"delegate_task", "clarify", "memory", "send_message", "execute_code"}
	for _, name := range want {
		if !BlockedTools[name] {
			t.Errorf("BlockedTools[%q]: want true, got false", name)
		}
	}
	if BlockedTools["echo"] {
		t.Errorf("BlockedTools[\"echo\"]: want false (real tool, not blocked), got true")
	}
}
