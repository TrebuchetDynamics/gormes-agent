package llm

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/events"
)

func TestTurnEvents_Lifecycle(t *testing.T) {
	bus := events.NewInProcessEventBus()
	defer bus.Close()
	emitter := NewTurnEventEmitter(bus)

	var count atomic.Int32
	for _, topic := range []string{TopicTurnStart, TopicTurnThought, TopicTurnAction, TopicTurnObserve, TopicTurnComplete} {
		bus.Subscribe(topic, func(e events.Event) { count.Add(1) })
	}

	emitter.EmitStart("test", "trace-1")
	emitter.EmitThought("test", "trace-1", "thinking about X")
	emitter.EmitAction("test", "trace-1", "echo", nil)
	emitter.EmitObserve("test", "trace-1", nil)
	emitter.EmitComplete("test", "trace-1")
	time.Sleep(30 * time.Millisecond)

	if count.Load() != 5 {
		t.Fatalf("emitted 5 events, received %d", count.Load())
	}
}

func TestTurnEvents_Error(t *testing.T) {
	bus := events.NewInProcessEventBus()
	defer bus.Close()
	emitter := NewTurnEventEmitter(bus)

	var count atomic.Int32
	bus.Subscribe(TopicTurnError, func(e events.Event) { count.Add(1) })

	emitter.EmitError("test", "trace-1", "something went wrong")
	time.Sleep(30 * time.Millisecond)

	if count.Load() != 1 {
		t.Fatalf("error event not received")
	}
}
