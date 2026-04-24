// internal/subagent/errors_test.go
package subagent

import (
	"errors"
	"fmt"
	"testing"
)

func TestSentinelIdentity(t *testing.T) {
	wrapped := fmt.Errorf("wrapped: %w", ErrMaxDepth)
	if !errors.Is(wrapped, ErrMaxDepth) {
		t.Errorf("errors.Is(wrapped, ErrMaxDepth): want true, got false")
	}
	if errors.Is(ErrMaxDepth, ErrSubagentNotFound) {
		t.Errorf("errors.Is(ErrMaxDepth, ErrSubagentNotFound): want false, got true (sentinels must be distinct)")
	}
}

func TestSentinelMessages(t *testing.T) {
	if ErrMaxDepth.Error() == "" || ErrSubagentNotFound.Error() == "" {
		t.Errorf("sentinel errors must have non-empty messages")
	}
}
