package subagent

import "testing"

func TestEventTypeStrings(t *testing.T) {
	cases := map[EventType]string{
		EventStarted:   "started",
		EventProgress:  "progress",
		EventToolCall:  "tool_call",
		EventCompleted: "completed",
		EventFailed:    "failed",
	}
	for got, want := range cases {
		if string(got) != want {
			t.Errorf("EventType = %q, want %q", got, want)
		}
	}
}

func TestResultStatusStrings(t *testing.T) {
	cases := map[ResultStatus]string{
		StatusCompleted: "completed",
		StatusFailed:    "failed",
		StatusCancelled: "cancelled",
		StatusTimedOut:  "timed_out",
	}
	for got, want := range cases {
		if string(got) != want {
			t.Errorf("ResultStatus = %q, want %q", got, want)
		}
	}
}
