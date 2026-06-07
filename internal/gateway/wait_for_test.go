package gateway

import (
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/gatewaytest"
)

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	gatewaytest.WaitFor(t, timeout, cond)
}
