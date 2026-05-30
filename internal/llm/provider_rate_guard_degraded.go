package llm

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/routing"
)

type GuardState = routing.GuardState

func ApplyClassification(state GuardState, now time.Time, class RateLimitClass) GuardState {
	return routing.ApplyClassification(state, now, class)
}
