package eventtest

import (
	"testing"
	"time"
)

// WaitUntil polls condition until it returns true or the timeout expires.
// It keeps asynchronous event tests tied to observable delivery instead of
// fixed sleeps.
func WaitUntil(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	if condition() {
		return
	}
	t.Fatalf("condition was not satisfied within %s", timeout)
}
