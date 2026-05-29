package signal

import (
	"context"
	"math"
	"reflect"
	"testing"
	"time"
)

func TestSignalAttachmentRateLimitInitialEstimateAndFeedback(t *testing.T) {
	clock := newFakeSignalClock()
	s := NewSignalAttachmentScheduler(WithSignalAttachmentSchedulerNow(clock.Now))

	if s.Capacity() != SignalRateLimitBucketCapacity {
		t.Fatalf("capacity = %d, want %d", s.Capacity(), SignalRateLimitBucketCapacity)
	}
	if s.Tokens() != float64(SignalRateLimitBucketCapacity) {
		t.Fatalf("tokens = %.1f, want full capacity", s.Tokens())
	}
	if got := s.EstimateWait(10); got != 0 {
		t.Fatalf("EstimateWait(10) = %s, want 0", got)
	}

	s.Feedback(42*time.Second, 1)

	if s.Tokens() != 0 {
		t.Fatalf("tokens after feedback = %.1f, want 0", s.Tokens())
	}
	if got := s.EstimateWait(1); got != 42*time.Second {
		t.Fatalf("EstimateWait(1) after feedback = %s, want 42s", got)
	}
	if math.Abs(s.RefillSecondsPerToken()-42) > 0.001 {
		t.Fatalf("RefillSecondsPerToken = %.3f, want 42", s.RefillSecondsPerToken())
	}
}

func TestSignalAttachmentRateLimitAcquireSleepsForDeficitAndReportDeducts(t *testing.T) {
	clock := newFakeSignalClock()
	var sleeps []time.Duration
	s := NewSignalAttachmentScheduler(
		WithSignalAttachmentSchedulerNow(clock.Now),
		WithSignalAttachmentSchedulerSleep(func(_ context.Context, d time.Duration) error {
			sleeps = append(sleeps, d)
			clock.Advance(d)
			return nil
		}),
	)
	s.setTokensForTest(0)

	waited, err := s.Acquire(context.Background(), 32)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if waited != 128*time.Second {
		t.Fatalf("Acquire waited = %s, want 128s", waited)
	}
	if !reflect.DeepEqual(sleeps, []time.Duration{128 * time.Second}) {
		t.Fatalf("sleeps = %v, want [128s]", sleeps)
	}

	s.ReportRPCDuration(time.Nanosecond, 32)
	if s.Tokens() != 0 {
		t.Fatalf("tokens after report = %.1f, want 0", s.Tokens())
	}
}

func TestSignalAttachmentRateLimitConcurrentAcquireSerializesFIFO(t *testing.T) {
	clock := newFakeSignalClock()
	var sleeps []time.Duration
	s := NewSignalAttachmentScheduler(
		WithSignalAttachmentSchedulerNow(clock.Now),
		WithSignalAttachmentSchedulerSleep(func(_ context.Context, d time.Duration) error {
			sleeps = append(sleeps, d)
			clock.Advance(d)
			return nil
		}),
	)

	type result struct {
		label string
		wait  time.Duration
		err   error
	}
	results := make(chan result, 2)
	startB := make(chan struct{})

	go func() {
		waited, err := s.Acquire(context.Background(), SignalRateLimitBucketCapacity)
		s.ReportRPCDuration(time.Nanosecond, SignalRateLimitBucketCapacity)
		close(startB)
		results <- result{label: "A", wait: waited, err: err}
	}()
	go func() {
		<-startB
		waited, err := s.Acquire(context.Background(), SignalRateLimitBucketCapacity)
		s.ReportRPCDuration(time.Nanosecond, SignalRateLimitBucketCapacity)
		results <- result{label: "B", wait: waited, err: err}
	}()

	first := <-results
	second := <-results
	if first.err != nil || second.err != nil {
		t.Fatalf("acquire errors: first=%v second=%v", first.err, second.err)
	}
	if first.label != "A" || first.wait != 0 {
		t.Fatalf("first result = %+v, want A/0", first)
	}
	if second.label != "B" || second.wait != 200*time.Second {
		t.Fatalf("second result = %+v, want B/200s", second)
	}
	if !reflect.DeepEqual(sleeps, []time.Duration{200 * time.Second}) {
		t.Fatalf("sleeps = %v, want [200s]", sleeps)
	}
}

func TestSignalAttachmentRateLimitSingletonReset(t *testing.T) {
	ResetSignalAttachmentSchedulerForTest()
	first := GetSignalAttachmentScheduler()
	second := GetSignalAttachmentScheduler()
	if first != second {
		t.Fatal("GetSignalAttachmentScheduler returned different instances")
	}
	ResetSignalAttachmentSchedulerForTest()
	third := GetSignalAttachmentScheduler()
	if third == first {
		t.Fatal("ResetSignalAttachmentSchedulerForTest reused previous instance")
	}
}

type fakeSignalClock struct {
	now time.Time
}

func newFakeSignalClock() *fakeSignalClock {
	return &fakeSignalClock{now: time.Unix(1000, 0)}
}

func (c *fakeSignalClock) Now() time.Time {
	return c.now
}

func (c *fakeSignalClock) Advance(d time.Duration) {
	c.now = c.now.Add(d)
}
