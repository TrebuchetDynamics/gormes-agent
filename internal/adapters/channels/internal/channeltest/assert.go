// Package channeltest provides reusable test helpers for channel adapter tests.
package channeltest

import (
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

// AssertNoInbound fails if inbox receives a gateway inbound event before the
// short quiet-period timeout used by channel policy/drop tests.
func AssertNoInbound(t testing.TB, inbox <-chan gateway.InboundEvent) {
	t.Helper()
	select {
	case ev := <-inbox:
		t.Fatalf("expected no inbound event, got %+v", ev)
	case <-time.After(50 * time.Millisecond):
	}
}
