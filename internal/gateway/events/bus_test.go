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

func TestEventBusCore_PubSubRoutingTypedEvent(t *testing.T) {
	bus := NewInProcessEventBus()
	defer bus.Close()

	delivered := make(chan Event, 1)
	bus.Subscribe("memory.updated", func(e Event) {
		delivered <- e
	})
	evt := NewEvent("memory.updated", "goncho", json.RawMessage(`{"id":"mem_1"}`), "trace-memory-1")
	if evt.Timestamp.IsZero() || evt.Source != "goncho" || evt.TraceID != "trace-memory-1" || string(evt.Payload) == "" {
		t.Fatalf("event provenance = %+v, want type/timestamp/source/payload/trace_id", evt)
	}
	if err := bus.Publish("memory.updated", evt); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	select {
	case got := <-delivered:
		if got.Type != evt.Type || got.Source != evt.Source || got.TraceID != evt.TraceID || string(got.Payload) != string(evt.Payload) {
			t.Fatalf("delivered event = %+v, want %+v", got, evt)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for typed event delivery")
	}
}

func TestEventBusCore_TopicIsolationAndUnsubscribe(t *testing.T) {
	bus := NewInProcessEventBus()
	defer bus.Close()

	var gotA, gotB atomic.Int32
	unsub := bus.Subscribe("gateway.message", func(Event) { gotA.Add(1) })
	bus.Subscribe("agent.turn", func(Event) { gotB.Add(1) })
	if err := bus.Publish("gateway.message", NewEvent("message", "gateway", nil, "trace-1")); err != nil {
		t.Fatalf("Publish gateway.message: %v", err)
	}
	time.Sleep(30 * time.Millisecond)
	if gotA.Load() != 1 || gotB.Load() != 0 {
		t.Fatalf("topic counts before unsubscribe = gateway:%d agent:%d, want 1/0", gotA.Load(), gotB.Load())
	}

	unsub()
	if err := bus.Publish("gateway.message", NewEvent("message", "gateway", nil, "trace-2")); err != nil {
		t.Fatalf("Publish gateway.message after unsubscribe: %v", err)
	}
	time.Sleep(30 * time.Millisecond)
	if gotA.Load() != 1 {
		t.Fatalf("gateway count after unsubscribe = %d, want unchanged 1", gotA.Load())
	}
}

func TestEventBusCore_BackpressureDoesNotBlockPublisher(t *testing.T) {
	bus := NewInProcessEventBus()
	defer bus.Close()

	blocker := make(chan struct{})
	bus.Subscribe("slow", func(Event) {
		<-blocker
	})

	start := time.Now()
	for i := 0; i < 100; i++ {
		if err := bus.Publish("slow", NewEvent("tick", "test", nil, "trace-slow")); err != nil {
			t.Fatalf("Publish slow: %v", err)
		}
	}
	elapsed := time.Since(start)
	close(blocker)
	if elapsed > 50*time.Millisecond {
		t.Fatalf("publishing to slow subscriber took %v, want nonblocking publish", elapsed)
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

func TestEventBusIntegration_MultipleProducersSubscribers(t *testing.T) {
	bus := NewInProcessEventBus()
	defer bus.Close()

	var count1, count2 atomic.Int32
	bus.Subscribe("gateway.msg", func(e Event) { count1.Add(1) })
	bus.Subscribe("agent.turn", func(e Event) { count2.Add(1) })

	bus.Publish("gateway.msg", NewEvent("msg", "telegram", nil, "t1"))
	bus.Publish("agent.turn", NewEvent("turn", "agent", nil, "t1"))
	time.Sleep(30 * time.Millisecond)

	if count1.Load() != 1 || count2.Load() != 1 {
		t.Fatalf("count1=%d count2=%d, want both=1", count1.Load(), count2.Load())
	}
}

func TestEventBusIntegration_MessageLifecycle(t *testing.T) {
	bus := NewInProcessEventBus()
	defer bus.Close()

	var received, sent, completed atomic.Int32
	bus.Subscribe("gateway.message.received", func(e Event) { received.Add(1) })
	bus.Subscribe("gateway.message.sent", func(e Event) { sent.Add(1) })
	bus.Subscribe("agent.turn.complete", func(e Event) { completed.Add(1) })

	bus.Publish("gateway.message.received", NewEvent("msg", "telegram", nil, "t1"))
	bus.Publish("agent.turn.complete", NewEvent("turn", "agent", nil, "t1"))
	bus.Publish("gateway.message.sent", NewEvent("msg", "telegram", nil, "t1"))
	time.Sleep(30 * time.Millisecond)

	if received.Load() != 1 || sent.Load() != 1 || completed.Load() != 1 {
		t.Fatalf("received=%d sent=%d completed=%d, want all=1", received.Load(), sent.Load(), completed.Load())
	}
}
