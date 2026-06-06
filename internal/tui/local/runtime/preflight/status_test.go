package preflight

import (
	"strings"
	"testing"
)

func TestDoctorTUIStatusReportsRemoteDegradedMode(t *testing.T) {
	got := DoctorStatus().Format()
	lower := strings.ToLower(got)
	for _, want := range []string{"native tui", "go-native bubble tea", "remote", "websocket"} {
		if !strings.Contains(lower, want) {
			t.Errorf("doctor TUI status missing %q:\n%s", want, got)
		}
	}
}
