package kernel

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

// RetryBudget implements the Route-B reconnect schedule from spec §9.2:
// 1s, 2s, 4s, 8s, 16s with +/-20% jitter, then exhausted. Not goroutine-safe;
// the kernel holds one budget per turn on the Run goroutine.
type RetryBudget struct {
	attempt int
}

const maxRetryAttempts = 5
const maxProviderRetryAfter = 16 * time.Second

// NewRetryBudget returns a fresh budget — 5 attempts remaining.
func NewRetryBudget() *RetryBudget { return &RetryBudget{} }

// NextDelay returns the jittered backoff for the next attempt, or -1 if the
// budget is exhausted. Advances the internal attempt counter on each call.
func (b *RetryBudget) NextDelay() time.Duration {
	if b.attempt >= maxRetryAttempts {
		return -1
	}
	b.attempt++
	base := time.Second << uint(b.attempt-1)
	jitter := rand.Float64()*0.4 - 0.2 // +/-0.2
	return time.Duration(float64(base) * (1.0 + jitter))
}

// NextDelayFor advances the retry budget and prefers a provider Retry-After
// hint when the triggering error carries one. The hint is capped to the
// reconnect budget's maximum base delay so a provider cannot stall the kernel
// beyond the bounded Route-B recovery window.
func (b *RetryBudget) NextDelayFor(err error) time.Duration {
	scheduled := b.NextDelay()
	if scheduled < 0 {
		return scheduled
	}
	hint := providerRetryAfter(err)
	if hint <= 0 {
		return scheduled
	}
	if hint > maxProviderRetryAfter {
		return maxProviderRetryAfter
	}
	return hint
}

// Exhausted returns true if NextDelay has been called maxRetryAttempts times.
func (b *RetryBudget) Exhausted() bool {
	return b.attempt >= maxRetryAttempts
}

// Wait sleeps for d or returns early on ctx cancellation. Returns ctx.Err()
// on cancellation, nil on clean timer expiration.
func Wait(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func providerRetryAfter(err error) time.Duration {
	var httpErr *hermes.HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.RetryAfter
	}
	return 0
}
