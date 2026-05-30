package resilience

import (
	"sync"
	"time"
)

// CircuitState represents the current state of a circuit breaker.
type CircuitState int

const (
	CircuitClosed   CircuitState = iota // normal operation, calls allowed
	CircuitOpen                         // failing, calls fast-rejected
	CircuitHalfOpen                     // probing recovery
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "CLOSED"
	case CircuitOpen:
		return "OPEN"
	case CircuitHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreaker tracks consecutive failures and opens the circuit when
// a threshold is exceeded. Each provider+API-key combination should have
// its own independent CircuitBreaker.
type CircuitBreaker struct {
	mu          sync.Mutex
	state       CircuitState
	failures    int
	threshold   int
	cooldown    time.Duration
	lastFailure time.Time
}

// NewCircuitBreaker creates a breaker that opens after threshold consecutive
// failures and stays open for at least cooldown before probing recovery.
func NewCircuitBreaker(threshold int, cooldown time.Duration) *CircuitBreaker {
	if threshold <= 0 {
		threshold = 5
	}
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	return &CircuitBreaker{
		state:     CircuitClosed,
		threshold: threshold,
		cooldown:  cooldown,
	}
}

// Allow reports whether a call should be attempted. Returns true when the
// circuit is CLOSED or HALF_OPEN. When OPEN, checks if the cooldown period
// has elapsed and transitions to HALF_OPEN if so.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(cb.lastFailure) >= cb.cooldown {
			cb.state = CircuitHalfOpen
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess reports a successful call. Resets the failure counter. In
// HALF_OPEN state, transitions back to CLOSED.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	if cb.state == CircuitHalfOpen {
		cb.state = CircuitClosed
	}
}

// RecordFailure reports a failed call. Increments the failure counter and
// opens the circuit if the threshold is reached. In HALF_OPEN, immediately
// transitions back to OPEN.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.failures >= cb.threshold || cb.state == CircuitHalfOpen {
		cb.state = CircuitOpen
	}
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// PerKeyCircuitBreakers manages independent CircuitBreaker instances keyed
// by a provider+API-key identifier.
type PerKeyCircuitBreakers struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
}

// NewPerKeyCircuitBreakers creates a new per-key breaker manager.
func NewPerKeyCircuitBreakers() *PerKeyCircuitBreakers {
	return &PerKeyCircuitBreakers{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// GetOrCreate returns the breaker for key, creating one with the given
// threshold and cooldown if it does not exist.
func (p *PerKeyCircuitBreakers) GetOrCreate(key string, threshold int, cooldown time.Duration) *CircuitBreaker {
	p.mu.RLock()
	cb, ok := p.breakers[key]
	p.mu.RUnlock()
	if ok {
		return cb
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	// double-check after acquiring write lock
	if cb, ok = p.breakers[key]; ok {
		return cb
	}
	cb = NewCircuitBreaker(threshold, cooldown)
	p.breakers[key] = cb
	return cb
}
