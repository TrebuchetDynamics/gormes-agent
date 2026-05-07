package signal

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

const (
	SignalMaxAttachmentsPerMessage      = 32
	SignalRateLimitBucketCapacity       = 50
	SignalRateLimitDefaultRetryAfterSec = 4
)

type SignalAttachmentScheduler struct {
	mu         sync.Mutex
	gate       chan struct{}
	capacity   float64
	tokens     float64
	refillRate float64
	lastRefill time.Time
	now        func() time.Time
	sleep      func(context.Context, time.Duration) error
}

type SignalAttachmentSchedulerOption func(*SignalAttachmentScheduler)

func WithSignalAttachmentSchedulerNow(now func() time.Time) SignalAttachmentSchedulerOption {
	return func(s *SignalAttachmentScheduler) {
		if now != nil {
			s.now = now
			s.lastRefill = now()
		}
	}
}

func WithSignalAttachmentSchedulerSleep(sleep func(context.Context, time.Duration) error) SignalAttachmentSchedulerOption {
	return func(s *SignalAttachmentScheduler) {
		if sleep != nil {
			s.sleep = sleep
		}
	}
}

func NewSignalAttachmentScheduler(opts ...SignalAttachmentSchedulerOption) *SignalAttachmentScheduler {
	now := time.Now
	s := &SignalAttachmentScheduler{
		gate:       make(chan struct{}, 1),
		capacity:   SignalRateLimitBucketCapacity,
		tokens:     SignalRateLimitBucketCapacity,
		refillRate: 1.0 / float64(SignalRateLimitDefaultRetryAfterSec),
		now:        now,
		sleep: func(ctx context.Context, d time.Duration) error {
			timer := time.NewTimer(d)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
	}
	s.lastRefill = now()
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *SignalAttachmentScheduler) Capacity() int {
	return int(s.capacity)
}

func (s *SignalAttachmentScheduler) Tokens() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refillLocked()
	return s.tokens
}

func (s *SignalAttachmentScheduler) RefillSecondsPerToken() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.refillRate <= 0 {
		return math.Inf(1)
	}
	return 1.0 / s.refillRate
}

func (s *SignalAttachmentScheduler) EstimateWait(n int) time.Duration {
	if n <= 0 {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refillLocked()
	return s.waitForTokensLocked(n)
}

func (s *SignalAttachmentScheduler) Acquire(ctx context.Context, n int) (time.Duration, error) {
	if n <= 0 {
		return 0, nil
	}
	if float64(n) > s.capacity {
		return 0, fmt.Errorf("signal attachment scheduler requested %d tokens above capacity %d", n, int(s.capacity))
	}
	select {
	case s.gate <- struct{}{}:
	case <-ctx.Done():
		return 0, ctx.Err()
	}
	defer func() { <-s.gate }()

	var total time.Duration
	for {
		s.mu.Lock()
		s.refillLocked()
		wait := s.waitForTokensLocked(n)
		s.mu.Unlock()
		if wait <= 0 {
			return total, nil
		}
		if err := s.sleep(ctx, wait); err != nil {
			return total, err
		}
		total += wait
	}
}

func (s *SignalAttachmentScheduler) ReportRPCDuration(_ time.Duration, n int) {
	if n <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens = math.Max(0, s.tokens-float64(n))
	s.lastRefill = s.now()
}

func (s *SignalAttachmentScheduler) Feedback(retryAfter time.Duration, _ int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if retryAfter > 0 {
		s.refillRate = 1.0 / retryAfter.Seconds()
	}
	s.tokens = 0
	s.lastRefill = s.now()
}

func (s *SignalAttachmentScheduler) refillLocked() {
	now := s.now()
	elapsed := now.Sub(s.lastRefill).Seconds()
	if elapsed > 0 && s.tokens < s.capacity {
		s.tokens = math.Min(s.capacity, s.tokens+elapsed*s.refillRate)
	}
	s.lastRefill = now
}

func (s *SignalAttachmentScheduler) waitForTokensLocked(n int) time.Duration {
	if float64(n) <= s.tokens {
		return 0
	}
	deficit := float64(n) - s.tokens
	if s.refillRate <= 0 {
		return time.Duration(math.MaxInt64)
	}
	return time.Duration(math.Ceil(deficit / s.refillRate * float64(time.Second)))
}

func (s *SignalAttachmentScheduler) setTokensForTest(tokens float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens = tokens
	s.lastRefill = s.now()
}

var (
	signalAttachmentSchedulerMu sync.Mutex
	signalAttachmentScheduler   *SignalAttachmentScheduler
)

func GetSignalAttachmentScheduler() *SignalAttachmentScheduler {
	signalAttachmentSchedulerMu.Lock()
	defer signalAttachmentSchedulerMu.Unlock()
	if signalAttachmentScheduler == nil {
		signalAttachmentScheduler = NewSignalAttachmentScheduler()
	}
	return signalAttachmentScheduler
}

func ResetSignalAttachmentSchedulerForTest() {
	signalAttachmentSchedulerMu.Lock()
	defer signalAttachmentSchedulerMu.Unlock()
	signalAttachmentScheduler = nil
}
