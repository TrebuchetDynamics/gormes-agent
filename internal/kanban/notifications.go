package kanban

import (
	"context"
	"sync"
	"time"
)

// DefaultNotifyThrottleRate is the minimum interval between notifications
// for the same board.
const DefaultNotifyThrottleRate = 1 * time.Minute

// ThrottledNotifySender wraps a NotifySender and rate-limits notifications
// to at most one per board per throttle interval. Notifications that arrive
// within the window for the same board are silently dropped.
type ThrottledNotifySender struct {
	inner      NotifySender
	mu         sync.Mutex
	lastNotify map[string]time.Time
	rate       time.Duration
	now        func() time.Time
}

// NewThrottledNotifySender creates a sender that wraps inner and allows at
// most one SendKanbanNotification per board per rate interval. If rate is
// zero or negative, DefaultNotifyThrottleRate is used.
func NewThrottledNotifySender(inner NotifySender, rate time.Duration) *ThrottledNotifySender {
	if rate <= 0 {
		rate = DefaultNotifyThrottleRate
	}
	return &ThrottledNotifySender{
		inner:      inner,
		lastNotify: make(map[string]time.Time),
		rate:       rate,
		now:        time.Now,
	}
}

// NewThrottledNotifySenderWithClock is like NewThrottledNotifySender but
// accepts a clock function for testing.
func NewThrottledNotifySenderWithClock(inner NotifySender, rate time.Duration, now func() time.Time) *ThrottledNotifySender {
	if rate <= 0 {
		rate = DefaultNotifyThrottleRate
	}
	return &ThrottledNotifySender{
		inner:      inner,
		lastNotify: make(map[string]time.Time),
		rate:       rate,
		now:        now,
	}
}

// SendKanbanNotification sends the notification unless the board was notified
// within the throttle window.
func (t *ThrottledNotifySender) SendKanbanNotification(ctx context.Context, delivery NotifyDelivery) error {
	t.mu.Lock()
	board := delivery.Board
	last, seen := t.lastNotify[board]
	now := t.now()
	if seen && now.Sub(last) < t.rate {
		t.mu.Unlock()
		return nil
	}
	t.lastNotify[board] = now
	t.mu.Unlock()
	return t.inner.SendKanbanNotification(ctx, delivery)
}

// Rate returns the configured rate limit duration.
func (t *ThrottledNotifySender) Rate() time.Duration { return t.rate }

// Reset clears all tracked board notification times.
func (t *ThrottledNotifySender) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastNotify = make(map[string]time.Time)
}
