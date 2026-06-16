package gatewaytest

import (
	"testing"
	"time"
)

// WaitFor polls cond until it succeeds or timeout elapses. Very small timeouts
// are raised to one second to avoid scheduler flakes in gateway async tests.
func WaitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	if timeout < time.Second {
		timeout = time.Second
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}
