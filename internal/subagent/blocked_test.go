// internal/subagent/blocked_test.go
package subagent

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
