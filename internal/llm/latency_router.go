package llm

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/routing"
)

type LatencyTracker = routing.LatencyTracker
type LatencyRouter = routing.LatencyRouter

func NewLatencyTracker(capacity int) *LatencyTracker {
	return routing.NewLatencyTracker(capacity)
}

func NewLatencyRouter(threshold time.Duration) *LatencyRouter {
	return routing.NewLatencyRouter(threshold)
}
