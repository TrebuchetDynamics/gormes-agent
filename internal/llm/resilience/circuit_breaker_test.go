package resilience

import (
	"sync"
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedToOpen(t *testing.T) {
	cb := NewCircuitBreaker(3, 100*time.Millisecond)

	if cb.State() != CircuitClosed {
		t.Fatalf("initial state = %s, want CLOSED", cb.State())
	}

	for i := 0; i < 3; i++ {
		if !cb.Allow() {
			t.Fatalf("iteration %d: Allow() = false in CLOSED state", i)
		}
		cb.RecordFailure()
	}

	if cb.State() != CircuitOpen {
		t.Fatalf("after %d failures, state = %s, want OPEN", 3, cb.State())
	}
}

func TestCircuitBreaker_OpenFastFails(t *testing.T) {
	cb := NewCircuitBreaker(1, 1*time.Second)
	cb.Allow()
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Fatal("breaker did not open after single failure")
	}

	for i := 0; i < 10; i++ {
		if cb.Allow() {
			t.Fatalf("iteration %d: Allow() returned true in OPEN state", i)
		}
	}

	if cb.State() != CircuitOpen {
		t.Fatalf("state changed to %s during OPEN fast-fails, want OPEN", cb.State())
	}
}

func TestCircuitBreaker_OpenToHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(1, 10*time.Millisecond)
	cb.Allow()
	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Fatal("breaker did not open")
	}

	time.Sleep(20 * time.Millisecond)

	allowed := cb.Allow()
	if !allowed {
		t.Fatal("Allow() returned false after cooldown, expected transition to HALF_OPEN")
	}
	if cb.State() != CircuitHalfOpen {
		t.Fatalf("after cooldown Allow, state = %s, want HALF_OPEN", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenToClosed(t *testing.T) {
	cb := NewCircuitBreaker(1, 10*time.Millisecond)
	cb.Allow()
	cb.RecordFailure()
	time.Sleep(20 * time.Millisecond)
	cb.Allow()

	cb.RecordSuccess()

	if cb.State() != CircuitClosed {
		t.Fatalf("after success in HALF_OPEN, state = %s, want CLOSED", cb.State())
	}

	if !cb.Allow() {
		t.Fatal("Allow() returned false after returning to CLOSED")
	}
}

func TestCircuitBreaker_HalfOpenToOpen(t *testing.T) {
	cb := NewCircuitBreaker(1, 10*time.Millisecond)
	cb.Allow()
	cb.RecordFailure()
	time.Sleep(20 * time.Millisecond)
	cb.Allow()

	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Fatalf("after failure in HALF_OPEN, state = %s, want OPEN", cb.State())
	}
}

func TestCircuitBreaker_PerKeyIsolation(t *testing.T) {
	mg := NewPerKeyCircuitBreakers()

	keyA := "providerA:abc12345"
	keyB := "providerB:xyz67890"

	cbA := mg.GetOrCreate(keyA, 1, 50*time.Millisecond)
	cbB := mg.GetOrCreate(keyB, 1, 50*time.Millisecond)

	cbA.Allow()
	cbA.RecordFailure()

	if cbA.State() != CircuitOpen {
		t.Fatal("breaker A did not open")
	}
	if cbB.State() != CircuitClosed {
		t.Fatalf("breaker B state = %s, want CLOSED (should be isolated from A)", cbB.State())
	}

	if !cbB.Allow() {
		t.Fatal("breaker B Allow() returned false while CLOSED and isolated")
	}
}

func TestCircuitBreaker_Concurrent(t *testing.T) {
	cb := NewCircuitBreaker(20, 100*time.Millisecond)

	var wg sync.WaitGroup
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				cb.Allow()
				cb.RecordSuccess()
			}
		}()
	}
	wg.Wait()

	if cb.State() != CircuitClosed {
		t.Fatalf("after concurrent success-only calls, state = %s, want CLOSED", cb.State())
	}
}

func TestCircuitBreaker_Configurable(t *testing.T) {
	cb := NewCircuitBreaker(2, 20*time.Millisecond)

	for i := 0; i < 2; i++ {
		cb.Allow()
		cb.RecordFailure()
	}

	if cb.State() != CircuitOpen {
		t.Fatalf("after %d failures (threshold=2), state = %s, want OPEN", 2, cb.State())
	}
}

func TestCircuitBreaker_DefaultThreshold(t *testing.T) {
	cb := NewCircuitBreaker(0, 50*time.Millisecond)

	for i := 0; i < 5; i++ {
		cb.Allow()
		cb.RecordFailure()
	}

	if cb.State() != CircuitOpen {
		t.Fatalf("after 5 failures with default threshold=5, state = %s, want OPEN", cb.State())
	}
}
