package events

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPublishSubscribe(t *testing.T) {
	bus := NewInProcessEventBus()
	defer bus.Close()

	var received atomic.Int32
	unsub := bus.Subscribe("test", func(e Event) {
		received.Add(1)
	})
	defer unsub()

	evt := NewEvent("test.type", "tests", json.RawMessage(`{"key":"val"}`), "trace-1")
	bus.Publish("test", evt)

	time.Sleep(50 * time.Millisecond)

	if received.Load() != 1 {
		t.Fatalf("received %d events, want 1", received.Load())
	}
}

func TestMultipleSubscribers(t *testing.T) {
	bus := NewInProcessEventBus()
	defer bus.Close()

	var count atomic.Int32
	for i := 0; i < 3; i++ {
		bus.Subscribe("topic", func(e Event) {
			count.Add(1)
		})
	}

	bus.Publish("topic", NewEvent("x", "tests", nil, "t1"))
	time.Sleep(50 * time.Millisecond)

	if count.Load() != 3 {
		t.Fatalf("3 subscribers received %d events total, want 3", count.Load())
	}
}

func TestTopicIsolation(t *testing.T) {
	bus := NewInProcessEventBus()
	defer bus.Close()

	var gotA, gotB atomic.Int32
	bus.Subscribe("A", func(e Event) { gotA.Add(1) })
	bus.Subscribe("B", func(e Event) { gotB.Add(1) })

	bus.Publish("A", NewEvent("a", "tests", nil, "t1"))
	time.Sleep(30 * time.Millisecond)

	if gotA.Load() != 1 {
		t.Fatalf("topic A subscriber received %d, want 1", gotA.Load())
	}
	if gotB.Load() != 0 {
		t.Fatalf("topic B subscriber received %d, want 0 (topic isolation)", gotB.Load())
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := NewInProcessEventBus()
	defer bus.Close()

	var count atomic.Int32
	unsub := bus.Subscribe("topic", func(e Event) { count.Add(1) })
	unsub()

	bus.Publish("topic", NewEvent("x", "tests", nil, "t1"))
	time.Sleep(50 * time.Millisecond)

	if count.Load() != 0 {
		t.Fatalf("received %d events after unsubscribe, want 0", count.Load())
	}
}

func TestBackpressure(t *testing.T) {
	bus := NewInProcessEventBus()
	defer bus.Close()

	blocker := make(chan struct{})
	var handled atomic.Int32

	bus.Subscribe("slow", func(e Event) {
		<-blocker
		handled.Add(1)
	})

	for i := 0; i < 100; i++ {
		bus.Publish("slow", NewEvent("x", "tests", nil, "t1"))
	}

	close(blocker)
	time.Sleep(100 * time.Millisecond)

	if handled.Load() == 0 {
		t.Fatal("no events handled even after unblocking")
	}
}

func TestClose(t *testing.T) {
	bus := NewInProcessEventBus()

	var count atomic.Int32
	bus.Subscribe("topic", func(e Event) { count.Add(1) })
	bus.Publish("topic", NewEvent("x", "tests", nil, "t1"))
	time.Sleep(30 * time.Millisecond)

	if count.Load() != 1 {
		t.Fatalf("before close: got %d, want 1", count.Load())
	}

	bus.Close()
	bus.Publish("topic", NewEvent("x", "tests", nil, "t2"))
	time.Sleep(30 * time.Millisecond)

	if count.Load() != 1 {
		t.Fatalf("after close: got %d, want 1 (no more delivery)", count.Load())
	}
}

func TestConcurrentPublish(t *testing.T) {
	bus := NewInProcessEventBus()
	defer bus.Close()

	var count atomic.Int32
	bus.Subscribe("topic", func(e Event) { count.Add(1) })

	var wg sync.WaitGroup
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				bus.Publish("topic", NewEvent("x", "tests", nil, "t1"))
			}
		}()
	}
	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	if count.Load() == 0 {
		t.Fatal("no events received from concurrent publishers")
	}
}
