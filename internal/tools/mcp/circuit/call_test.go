package circuit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/callresult"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/content"
)

type fakeMCPClock struct {
	now time.Time
}

func (c *fakeMCPClock) Now() time.Time {
	return c.now
}

func (c *fakeMCPClock) Advance(d time.Duration) {
	c.now = c.now.Add(d)
}

func newTestBreaker(clock *fakeMCPClock) *Breaker {
	return NewBreaker(BreakerOptions{
		Threshold: 2,
		Cooldown:  time.Minute,
		Now:       clock.Now,
	})
}

func TestCallWithBreakerShortCircuitsBeforeCooldown(t *testing.T) {
	clock := &fakeMCPClock{now: time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)}
	breaker := newTestBreaker(clock)
	calls := 0
	call := func(context.Context) (callresult.Result, error) {
		calls++
		return callresult.Result{}, errors.New("server still broken")
	}

	for i := 0; i < 2; i++ {
		_, evidence, err := CallWithBreaker(context.Background(), breaker, "srv", call)
		if err == nil {
			t.Fatalf("call %d err = nil, want failure", i+1)
		}
		if evidence != EvidenceServerUnreachable {
			t.Fatalf("call %d evidence = %q, want %q", i+1, evidence, EvidenceServerUnreachable)
		}
	}

	_, evidence, err := CallWithBreaker(context.Background(), breaker, "srv", call)
	if !errors.Is(err, ErrBreakerOpen) {
		t.Fatalf("short-circuit err = %v, want ErrBreakerOpen", err)
	}
	if evidence != EvidenceBreakerOpen {
		t.Fatalf("short-circuit evidence = %q, want %q", evidence, EvidenceBreakerOpen)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2; breaker must not touch the session before cooldown", calls)
	}
}

func TestCallWithBreakerHalfOpenSuccessCloses(t *testing.T) {
	clock := &fakeMCPClock{now: time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)}
	breaker := newTestBreaker(clock)
	breaker.RecordFailure("srv", errors.New("first"))
	breaker.RecordFailure("srv", errors.New("second"))
	clock.Advance(time.Minute + time.Second)
	calls := 0

	_, evidence, err := CallWithBreaker(context.Background(), breaker, "srv", func(context.Context) (callresult.Result, error) {
		calls++
		return callresult.Result{Content: []content.Structured{{Kind: "text", Text: "ok"}}}, nil
	})
	if err != nil {
		t.Fatalf("half-open call err = %v, want nil", err)
	}
	if evidence != EvidenceOK {
		t.Fatalf("half-open success evidence = %q, want %q", evidence, EvidenceOK)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want one half-open probe", calls)
	}
	if got := breaker.ErrorCount("srv"); got != 0 {
		t.Fatalf("breaker count = %d, want 0 after successful probe", got)
	}
}

func TestCallWithBreakerHalfOpenFailureReopens(t *testing.T) {
	clock := &fakeMCPClock{now: time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)}
	breaker := newTestBreaker(clock)
	breaker.RecordFailure("srv", errors.New("first"))
	breaker.RecordFailure("srv", errors.New("second"))
	clock.Advance(time.Minute + time.Second)
	calls := 0
	call := func(context.Context) (callresult.Result, error) {
		calls++
		return callresult.Result{}, errors.New("probe failed")
	}

	_, evidence, err := CallWithBreaker(context.Background(), breaker, "srv", call)
	if err == nil {
		t.Fatal("half-open failure err = nil, want failure")
	}
	if evidence != EvidenceHalfOpenFailed {
		t.Fatalf("half-open failure evidence = %q, want %q", evidence, EvidenceHalfOpenFailed)
	}

	_, evidence, err = CallWithBreaker(context.Background(), breaker, "srv", call)
	if !errors.Is(err, ErrBreakerOpen) {
		t.Fatalf("post-failure err = %v, want ErrBreakerOpen", err)
	}
	if evidence != EvidenceBreakerOpen {
		t.Fatalf("post-failure evidence = %q, want %q", evidence, EvidenceBreakerOpen)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want only the half-open probe to touch the session", calls)
	}
}

func TestResetAfterReconnectClearsBreakerOnRecoveredOAuth(t *testing.T) {
	clock := &fakeMCPClock{now: time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)}
	breaker := newTestBreaker(clock)
	breaker.RecordFailure("srv", errors.New("first"))
	breaker.RecordFailure("srv", errors.New("second"))
	breaker.RecordFailure("srv", errors.New("third"))

	if evidence := breaker.ResetAfterReconnect("srv"); evidence != EvidenceReconnectReset {
		t.Fatalf("ResetAfterReconnect evidence = %q, want %q", evidence, EvidenceReconnectReset)
	}

	calls := 0
	call := func(context.Context) (callresult.Result, error) {
		calls++
		return callresult.Result{}, errors.New("retry still failed")
	}
	_, evidence, err := CallWithBreaker(context.Background(), breaker, "srv", call)
	if err == nil {
		t.Fatal("post-reconnect retry err = nil, want failure")
	}
	if evidence != EvidenceServerUnreachable {
		t.Fatalf("post-reconnect evidence = %q, want %q", evidence, EvidenceServerUnreachable)
	}
	if got := breaker.ErrorCount("srv"); got != 1 {
		t.Fatalf("post-reconnect count = %d, want fresh count 1", got)
	}

	_, _, _ = CallWithBreaker(context.Background(), breaker, "srv", call)
	if calls != 2 {
		t.Fatalf("calls = %d, want retry failure below threshold to leave the next call open", calls)
	}
}
