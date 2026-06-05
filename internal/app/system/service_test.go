package system

import (
	"testing"

	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestParseEventMode(t *testing.T) {
	cases := []struct {
		in   string
		want toolspkg.SystemEventMode
	}{
		{"", toolspkg.SystemEventModeNextHeartbeat},
		{" next-heartbeat ", toolspkg.SystemEventModeNextHeartbeat},
		{"now", toolspkg.SystemEventModeNow},
	}
	for _, tc := range cases {
		got, err := ParseEventMode(tc.in)
		if err != nil {
			t.Fatalf("ParseEventMode(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("ParseEventMode(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	if _, err := ParseEventMode("later"); err == nil || err.Error() != "system event: --mode must be now or next-heartbeat" {
		t.Fatalf("ParseEventMode(later) err = %v", err)
	}
}

func TestFirstDegradedMessage(t *testing.T) {
	if got := FirstDegradedMessage(nil); got != "system_unavailable" {
		t.Fatalf("empty = %q", got)
	}
	if got := FirstDegradedMessage([]toolspkg.SystemDegradedStatus{{Message: "message", Reason: "reason", Code: "code"}}); got != "message" {
		t.Fatalf("message precedence = %q", got)
	}
	if got := FirstDegradedMessage([]toolspkg.SystemDegradedStatus{{Reason: "reason", Code: "code"}}); got != "reason" {
		t.Fatalf("reason precedence = %q", got)
	}
	if got := FirstDegradedMessage([]toolspkg.SystemDegradedStatus{{Code: "code"}}); got != "code" {
		t.Fatalf("code fallback = %q", got)
	}
}
