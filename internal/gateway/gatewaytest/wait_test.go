package gatewaytest

import (
	"testing"
	"time"
)

func TestWaitForReturnsWhenConditionPasses(t *testing.T) {
	calls := 0
	WaitFor(t, time.Second, func() bool {
		calls++
		return calls == 2
	})
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}
