package routing

import (
	"time"

	latencypolicy "github.com/TrebuchetDynamics/gormes-agent/internal/llm/routing/latency"
)

type LatencyTracker = latencypolicy.LatencyTracker
type LatencyRouter = latencypolicy.LatencyRouter

func NewLatencyTracker(capacity int) *LatencyTracker {
	return latencypolicy.NewLatencyTracker(capacity)
}

func NewLatencyRouter(threshold time.Duration) *LatencyRouter {
	return latencypolicy.NewLatencyRouter(threshold)
}
