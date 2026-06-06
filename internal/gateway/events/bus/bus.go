package bus

import (
	"log/slog"
	"sync"

	eventcontract "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/events/contract"
)

const defaultSubscriberBufferSize = 64

// EventBus decouples publishers from subscribers via topic-based routing.
// Implementations must be safe for concurrent use.
type EventBus = eventcontract.EventBus

type subscriber struct {
	handler  EventHandler
	buffer   chan Event
	done     chan struct{}
	stopOnce sync.Once
}

// InProcessEventBus is an in-memory, topic-based pub/sub bus with no
// external dependencies. Safe for concurrent use.
type InProcessEventBus struct {
	mu         sync.RWMutex
	subs       map[string][]*subscriber
	closed     bool
	bufferSize int
	log        *slog.Logger
}

// NewInProcessEventBus creates a new in-process event bus with the default
// subscriber buffer size.
func NewInProcessEventBus() *InProcessEventBus {
	return &InProcessEventBus{
		subs:       make(map[string][]*subscriber),
		bufferSize: defaultSubscriberBufferSize,
		log:        slog.Default(),
	}
}

// Publish sends event to all subscribers of topic. If a subscriber's buffer
// is full, the event is dropped and a warning is logged. Never blocks.
func (b *InProcessEventBus) Publish(topic string, event Event) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.closed {
		return nil
	}

	subs, ok := b.subs[topic]
	if !ok {
		return nil
	}

	for _, s := range subs {
		select {
		case s.buffer <- event:
		default:
			b.log.Warn("event bus subscriber buffer full, dropping event",
				"topic", topic, "type", event.Type)
		}
	}
	return nil
}

// Subscribe registers handler for topic and starts a goroutine that delivers
// events. Returns an unsubscribe function.
func (b *InProcessEventBus) Subscribe(topic string, handler EventHandler) func() {
	if handler == nil {
		return func() {}
	}
	b.mu.Lock()
	b.ensureInitializedLocked()
	s := &subscriber{
		handler: handler,
		buffer:  make(chan Event, b.bufferSize),
		done:    make(chan struct{}, 1),
	}
	b.subs[topic] = append(b.subs[topic], s)
	b.mu.Unlock()

	go func() {
		for {
			select {
			case event := <-s.buffer:
				s.handler(event)
			case <-s.done:
				return
			}
		}
	}()

	return func() {
		s.stopOnce.Do(func() { close(s.done) })
		b.mu.Lock()
		defer b.mu.Unlock()
		subs := b.subs[topic]
		for i, sub := range subs {
			if sub == s {
				b.subs[topic] = append(subs[:i], subs[i+1:]...)
				if len(b.subs[topic]) == 0 {
					delete(b.subs, topic)
				}
				return
			}
		}
	}
}

// Close signals all subscriber goroutines to stop and clears subscriptions.
func (b *InProcessEventBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.closed = true
	for _, subs := range b.subs {
		for _, s := range subs {
			s.stopOnce.Do(func() { close(s.done) })
		}
	}
	b.subs = make(map[string][]*subscriber)
	return nil
}

func (b *InProcessEventBus) ensureInitializedLocked() {
	if b.subs == nil {
		b.subs = make(map[string][]*subscriber)
	}
	if b.bufferSize <= 0 {
		b.bufferSize = defaultSubscriberBufferSize
	}
	if b.log == nil {
		b.log = slog.Default()
	}
}
