package react

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/safety/plangate"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/safety/toolgate"
)

func newDefaultTestAdapter() *SafetyAdapter {
	return NewSafetyAdapter(plangate.NewDefaultGate(), toolgate.NewDefaultGate())
}
