package gateway

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/gatewaytest"
)

func assertContainsAll(t *testing.T, got string, wants ...string) {
	t.Helper()
	gatewaytest.AssertContainsAll(t, got, wants...)
}
