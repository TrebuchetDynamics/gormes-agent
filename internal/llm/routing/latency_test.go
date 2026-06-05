package routing

import (
	"testing"
	"time"
)

func TestLatencyTracker_P95(t *testing.T) {
	lt := NewLatencyTracker(10)

	for i := 0; i < 10; i++ {
		lt.Record(time.Duration(i*100) * time.Millisecond)
	}

	p95 := lt.P95()
	expected := 900 * time.Millisecond
	if p95 != expected {
		t.Fatalf("P95 = %v, want %v (10 samples 0-900ms)", p95, expected)
	}
}

func TestLatencyTracker_SmallSample(t *testing.T) {
	lt := NewLatencyTracker(10)
	lt.Record(200 * time.Millisecond)
	lt.Record(100 * time.Millisecond)

	p95 := lt.P95()
	if p95 != 200*time.Millisecond {
		t.Fatalf("P95 with 2 samples = %v, want 200ms", p95)
	}
}

func TestLatencyRouter_SelectHealthy(t *testing.T) {
	router := NewLatencyRouter(1 * time.Second)

	router.Record("fast", 50*time.Millisecond)
	router.Record("fast", 60*time.Millisecond)
	router.Record("fast", 55*time.Millisecond)
	router.Record("fast", 50*time.Millisecond)
	router.Record("fast", 55*time.Millisecond)

	router.Record("slow", 3*time.Second)
	router.Record("slow", 4*time.Second)
	router.Record("slow", 3*time.Second)
	router.Record("slow", 5*time.Second)
	router.Record("slow", 4*time.Second)

	selected := router.SelectProvider([]string{"slow", "fast"})
	if selected != "fast" {
		t.Fatalf("selected %q, want fast (low latency)", selected)
	}
}

func TestLatencyRouter_Degraded(t *testing.T) {
	router := NewLatencyRouter(100 * time.Millisecond)

	router.Record("provider", 500*time.Millisecond)
	router.Record("provider", 600*time.Millisecond)
	router.Record("provider", 550*time.Millisecond)
	router.Record("provider", 500*time.Millisecond)
	router.Record("provider", 600*time.Millisecond)

	if !router.IsDegraded("provider") {
		t.Fatal("provider with P95 > 100ms should be degraded")
	}
}

func TestLatencyRouter_NotDegraded(t *testing.T) {
	router := NewLatencyRouter(1 * time.Second)

	router.Record("provider", 50*time.Millisecond)
	router.Record("provider", 60*time.Millisecond)
	router.Record("provider", 55*time.Millisecond)
	router.Record("provider", 50*time.Millisecond)
	router.Record("provider", 55*time.Millisecond)

	if router.IsDegraded("provider") {
		t.Fatal("provider with P95 < 1s should not be degraded")
	}
}
