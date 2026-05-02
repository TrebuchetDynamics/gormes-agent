package gateway

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/events"
)

func TestEventDispatcher_PublishMessageReceived(t *testing.T) {
	bus := events.NewInProcessEventBus()
	defer bus.Close()

	var received atomic.Int32
	bus.Subscribe(TopicMessageReceived, func(e events.Event) {
		received.Add(1)
	})

	disp := NewEventDispatcher(bus)
	err := disp.PublishMessageReceived("telegram", "trace-1", map[string]string{"text": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(30 * time.Millisecond)

	if received.Load() != 1 {
		t.Fatalf("received %d events, want 1", received.Load())
	}
}

func TestEventDispatcher_PublishMessageSent(t *testing.T) {
	bus := events.NewInProcessEventBus()
	defer bus.Close()

	var received atomic.Int32
	bus.Subscribe(TopicMessageSent, func(e events.Event) {
		received.Add(1)
	})

	disp := NewEventDispatcher(bus)
	disp.PublishMessageSent("discord", "trace-2", map[string]string{"text": "reply"})
	time.Sleep(30 * time.Millisecond)

	if received.Load() != 1 {
		t.Fatalf("received %d events, want 1", received.Load())
	}
}

func TestEventDispatcher_SessionLifecycle(t *testing.T) {
	bus := events.NewInProcessEventBus()
	defer bus.Close()

	var started, ended atomic.Int32
	bus.Subscribe(TopicSessionStarted, func(e events.Event) { started.Add(1) })
	bus.Subscribe(TopicSessionEnded, func(e events.Event) { ended.Add(1) })

	disp := NewEventDispatcher(bus)
	disp.PublishSessionStarted("slack", "trace-3", map[string]string{"user": "alice"})
	disp.PublishSessionEnded("slack", "trace-3", map[string]string{"user": "alice"})
	time.Sleep(30 * time.Millisecond)

	if started.Load() != 1 || ended.Load() != 1 {
		t.Fatalf("started=%d ended=%d, want 1 each", started.Load(), ended.Load())
	}
}

func TestEventDispatcher_SubscribeMessages(t *testing.T) {
	bus := events.NewInProcessEventBus()
	defer bus.Close()

	disp := NewEventDispatcher(bus)

	var count atomic.Int32
	unsub := disp.SubscribeMessages(func(e events.Event) {
		count.Add(1)
	})

	disp.PublishMessageReceived("telegram", "t1", map[string]string{"x": "y"})
	time.Sleep(30 * time.Millisecond)
	if count.Load() != 1 {
		t.Fatal("subscriber did not receive event")
	}

	unsub()
	disp.PublishMessageReceived("telegram", "t2", map[string]string{"x": "y"})
	time.Sleep(30 * time.Millisecond)
	if count.Load() != 1 {
		t.Fatal("unsubscribed handler received event")
	}
}
