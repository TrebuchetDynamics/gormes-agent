package tools

import (
	"context"
	"sync"
	"testing"
	"time"
)

// fakeClock implements a simple controllable clock for testing.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock(t time.Time) *fakeClock {
	return &fakeClock{now: t}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// fakeSessionBackend records close calls for test assertions.
type fakeSessionBackend struct {
	mu       sync.Mutex
	closed   []string
	closeErr error
}

func (f *fakeSessionBackend) Close(ctx context.Context, sessionID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = append(f.closed, sessionID)
	return f.closeErr
}

func (f *fakeSessionBackend) ClosedSessions() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.closed))
	copy(out, f.closed)
	return out
}

type reentrantSessionBackend struct {
	onClose func()
}

func (b *reentrantSessionBackend) Close(context.Context, string) error {
	if b.onClose != nil {
		b.onClose()
	}
	return nil
}

func TestBrowserInactivityCleanup_ReapsIdleSession(t *testing.T) {
	anchor := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	clock := newFakeClock(anchor)
	backend := &fakeSessionBackend{}

	tracker := NewBrowserSessionTracker(clock.Now)
	tracker.Register("sess-idle", backend, anchor.Add(-350*time.Second))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reaped := tracker.reapInactive(clock.Now(), 300*time.Second)
	if len(reaped) != 1 {
		t.Fatalf("expected 1 reaped session, got %d", len(reaped))
	}
	if reaped[0].SessionID != "sess-idle" {
		t.Errorf("expected sess-idle, got %s", reaped[0].SessionID)
	}
	closed := backend.ClosedSessions()
	if len(closed) != 1 || closed[0] != "sess-idle" {
		t.Errorf("expected sess-idle to be closed, got %v", closed)
	}
	// Session should be removed from tracker after reaping.
	if tracker.Len() != 0 {
		t.Errorf("expected 0 sessions after reap, got %d", tracker.Len())
	}

	_ = ctx // used for cleanup goroutine tests below
}

func TestBrowserInactivityCleanup_KeepsActiveSession(t *testing.T) {
	anchor := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	clock := newFakeClock(anchor)
	backend := &fakeSessionBackend{}

	tracker := NewBrowserSessionTracker(clock.Now)
	tracker.Register("sess-active", backend, anchor.Add(-30*time.Second))

	reaped := tracker.reapInactive(clock.Now(), 300*time.Second)
	if len(reaped) != 0 {
		t.Fatalf("expected 0 reaped sessions, got %d", len(reaped))
	}
	if tracker.Len() != 1 {
		t.Errorf("expected 1 session remaining, got %d", tracker.Len())
	}
}

func TestBrowserInactivityCleanup_ReapInterval(t *testing.T) {
	anchor := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	clock := newFakeClock(anchor)
	backend := &fakeSessionBackend{}

	tracker := NewBrowserSessionTracker(clock.Now)
	tracker.Register("sess-1", backend, anchor.Add(-400*time.Second))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanupDone := make(chan struct{})
	go func() {
		tracker.RunCleanup(ctx, clock.Now, 100*time.Millisecond, 300*time.Second)
		close(cleanupDone)
	}()

	// First cycle: session is idle (400s > 300s), should be reaped.
	time.Sleep(150 * time.Millisecond)
	if tracker.Len() != 0 {
		t.Errorf("after first cycle: expected 0 sessions, got %d", tracker.Len())
	}

	// Register a second session that is active (just touched).
	tracker.Register("sess-2", backend, clock.Now())

	// Advance past interval but not past inactivity timeout.
	clock.Advance(150 * time.Second)

	// Second cycle: session is active (150s < 300s), should NOT be reaped.
	time.Sleep(150 * time.Millisecond)
	if tracker.Len() != 1 {
		t.Errorf("after second cycle: expected 1 session, got %d", tracker.Len())
	}

	cancel()
	<-cleanupDone
}

func TestBrowserInactivityCleanup_EnvOverride(t *testing.T) {
	anchor := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	clock := newFakeClock(anchor)
	backend := &fakeSessionBackend{}

	t.Setenv("GORMES_BROWSER_INACTIVITY_TIMEOUT", "120")

	tracker := NewBrowserSessionTracker(clock.Now)
	tracker.Register("sess-idle-150", backend, anchor.Add(-150*time.Second))

	reaped := tracker.reapInactive(clock.Now(), tracker.inactivityTimeout())
	if len(reaped) != 1 {
		t.Fatalf("with timeout=120s and idle=150s, expected 1 reaped session, got %d", len(reaped))
	}
	if reaped[0].SessionID != "sess-idle-150" {
		t.Errorf("expected sess-idle-150, got %s", reaped[0].SessionID)
	}
}

func TestBrowserInactivityCleanup_NilBackendNoPanic(t *testing.T) {
	anchor := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	clock := newFakeClock(anchor)

	tracker := NewBrowserSessionTracker(clock.Now)
	tracker.Register("sess-nil-backend", nil, anchor.Add(-400*time.Second))

	reaped := tracker.reapInactive(clock.Now(), 300*time.Second)
	// Session with nil backend should still be reaped from the registry
	// (the backend Close is skipped, but the entry is removed).
	if len(reaped) != 1 {
		t.Fatalf("expected 1 reaped entry, got %d", len(reaped))
	}
	if tracker.Len() != 0 {
		t.Errorf("expected 0 sessions after reap of nil-backend session, got %d", tracker.Len())
	}
}

func TestBrowserInactivityCleanup_DoesNotHoldTrackerLockDuringBackendClose(t *testing.T) {
	anchor := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	clock := newFakeClock(anchor)

	tracker := NewBrowserSessionTracker(clock.Now)
	backend := &reentrantSessionBackend{
		onClose: func() {
			_ = tracker.Len()
		},
	}
	tracker.Register("sess-reentrant", backend, anchor.Add(-350*time.Second))

	done := make(chan []ReapEntry, 1)
	go func() {
		done <- tracker.reapInactive(clock.Now(), 300*time.Second)
	}()

	select {
	case reaped := <-done:
		if len(reaped) != 1 || reaped[0].SessionID != "sess-reentrant" {
			t.Fatalf("reaped = %+v, want sess-reentrant", reaped)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("reapInactive deadlocked while backend Close re-entered tracker")
	}
}

func TestBrowserInactivityCleanup_ReapFailureEvidence(t *testing.T) {
	anchor := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	clock := newFakeClock(anchor)

	failingBackend := &fakeSessionBackend{closeErr: errStubCloseFailed}
	okBackend := &fakeSessionBackend{}

	tracker := NewBrowserSessionTracker(clock.Now)
	tracker.Register("sess-fail", failingBackend, anchor.Add(-350*time.Second))
	tracker.Register("sess-ok", okBackend, anchor.Add(-350*time.Second))

	reaped := tracker.reapInactive(clock.Now(), 300*time.Second)
	if len(reaped) != 2 {
		t.Fatalf("expected 2 reaped entries, got %d", len(reaped))
	}

	// sess-fail should have an error recorded.
	var failEntry *ReapEntry
	var okEntry *ReapEntry
	for i := range reaped {
		if reaped[i].SessionID == "sess-fail" {
			failEntry = &reaped[i]
		}
		if reaped[i].SessionID == "sess-ok" {
			okEntry = &reaped[i]
		}
	}
	if failEntry == nil || failEntry.Err == nil {
		t.Errorf("expected sess-fail to have a non-nil error, got %v", failEntry)
	}
	if okEntry == nil || okEntry.Err != nil {
		t.Errorf("expected sess-ok to have no error, got %v", okEntry)
	}

	// Both sessions should still be removed from tracker.
	if tracker.Len() != 0 {
		t.Errorf("expected 0 sessions after reap (both reaped), got %d", tracker.Len())
	}

	// okBackend should have been called.
	if len(okBackend.ClosedSessions()) != 1 {
		t.Errorf("expected okBackend to have 1 close call, got %d", len(okBackend.ClosedSessions()))
	}
}

var errStubCloseFailed = &stubCloseFailedError{}

type stubCloseFailedError struct{}

func (e *stubCloseFailedError) Error() string { return "stub: close failed" }
