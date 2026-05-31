package latency

import (
	"math"
	"sort"
	"sync"
	"time"
)

const defaultLatencyWindow = 100

type LatencyTracker struct {
	mu        sync.Mutex
	latencies []time.Duration
	capacity  int
	index     int
	count     int
}

func NewLatencyTracker(capacity int) *LatencyTracker {
	if capacity <= 0 {
		capacity = defaultLatencyWindow
	}
	return &LatencyTracker{
		latencies: make([]time.Duration, capacity),
		capacity:  capacity,
	}
}

func (lt *LatencyTracker) Record(d time.Duration) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	lt.latencies[lt.index] = d
	lt.index = (lt.index + 1) % lt.capacity
	if lt.count < lt.capacity {
		lt.count++
	}
}

func (lt *LatencyTracker) P95() time.Duration {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	if lt.count == 0 {
		return 0
	}
	sorted := make([]time.Duration, lt.count)
	copy(sorted, lt.latencies[:lt.count])
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	idx := int(math.Ceil(float64(lt.count)*0.95)) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= lt.count {
		idx = lt.count - 1
	}
	return sorted[idx]
}

func (lt *LatencyTracker) Count() int {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	return lt.count
}

type LatencyRouter struct {
	mu            sync.RWMutex
	trackers      map[string]*LatencyTracker
	threshold     time.Duration
	degradeWeight float64
}

func NewLatencyRouter(threshold time.Duration) *LatencyRouter {
	if threshold <= 0 {
		threshold = 5 * time.Second
	}
	return &LatencyRouter{
		trackers:      make(map[string]*LatencyTracker),
		threshold:     threshold,
		degradeWeight: 0.1,
	}
}

func (r *LatencyRouter) Record(provider string, d time.Duration) {
	r.mu.Lock()
	lt, ok := r.trackers[provider]
	if !ok {
		lt = NewLatencyTracker(defaultLatencyWindow)
		r.trackers[provider] = lt
	}
	r.mu.Unlock()
	lt.Record(d)
}

func (r *LatencyRouter) SelectProvider(candidates []string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var best string
	var bestScore float64 = -1

	for _, p := range candidates {
		lt, ok := r.trackers[p]
		if !ok || lt.Count() < 5 {
			return p
		}
		p95 := lt.P95()
		score := 1.0
		if p95 > r.threshold {
			score = r.degradeWeight
		}
		if score > bestScore {
			bestScore = score
			best = p
		}
	}
	return best
}

func (r *LatencyRouter) IsDegraded(provider string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	lt, ok := r.trackers[provider]
	if !ok || lt.Count() < 5 {
		return false
	}
	return lt.P95() > r.threshold
}
