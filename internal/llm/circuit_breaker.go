package llm

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/resilience"
)

// CircuitState represents the current state of a circuit breaker.
type CircuitState = resilience.CircuitState

const (
	CircuitClosed   = resilience.CircuitClosed
	CircuitOpen     = resilience.CircuitOpen
	CircuitHalfOpen = resilience.CircuitHalfOpen
)

// CircuitBreaker tracks consecutive failures and opens the circuit when
// a threshold is exceeded.
type CircuitBreaker = resilience.CircuitBreaker

// NewCircuitBreaker creates a breaker that opens after threshold consecutive
// failures and stays open for at least cooldown before probing recovery.
func NewCircuitBreaker(threshold int, cooldown time.Duration) *CircuitBreaker {
	return resilience.NewCircuitBreaker(threshold, cooldown)
}

// PerKeyCircuitBreakers manages independent CircuitBreaker instances keyed
// by a provider+API-key identifier.
type PerKeyCircuitBreakers = resilience.PerKeyCircuitBreakers

// NewPerKeyCircuitBreakers creates a new per-key breaker manager.
func NewPerKeyCircuitBreakers() *PerKeyCircuitBreakers {
	return resilience.NewPerKeyCircuitBreakers()
}
