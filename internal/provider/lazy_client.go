// Package provider owns the lazy client pool used for provider HTTP client
// construction. It defers client instantiation until a code path actually
// selects a provider, avoiding eager construction of unselected providers
// during cold start.
package provider

import (
	"fmt"
	"sync"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

// ClientFactory constructs a hermes.Client for a named provider.
type ClientFactory func() (hermes.Client, error)

// ClientPool holds registered provider factories and lazily constructs
// clients on first access. It is safe for concurrent use.
type ClientPool struct {
	mu        sync.Mutex
	factories map[string]ClientFactory
	clients   map[string]hermes.Client
}

// NewClientPool returns an empty ClientPool ready for registration.
func NewClientPool() *ClientPool {
	return &ClientPool{
		factories: make(map[string]ClientFactory),
		clients:   make(map[string]hermes.Client),
	}
}

// Register adds a provider factory to the pool. It must be called before
// any Get for that provider. Registering the same provider twice replaces
// the previous factory but does not affect an already-constructed client.
func (p *ClientPool) Register(name string, factory ClientFactory) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.factories[name] = factory
}

// Get returns the hermes.Client for the named provider, constructing it
// via the registered factory on first access. Subsequent calls return the
// same instance. It returns an error if the provider is not registered or
// if the factory fails.
func (p *ClientPool) Get(name string) (hermes.Client, error) {
	// Fast path: client already constructed.
	p.mu.Lock()
	c, ok := p.clients[name]
	p.mu.Unlock()
	if ok {
		return c, nil
	}

	// Slow path: construct under lock.
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring the lock.
	if c, ok := p.clients[name]; ok {
		return c, nil
	}

	factory, ok := p.factories[name]
	if !ok {
		return nil, fmt.Errorf("provider %q not registered", name)
	}

	client, err := factory()
	if err != nil {
		return nil, fmt.Errorf("provider %q: %w", name, err)
	}

	p.clients[name] = client
	return client, nil
}

// Reset clears all constructed clients so subsequent Get calls re-invoke
// the factory. This is intended for test and benchmark use only; it must
// not be called from production code paths.
func (p *ClientPool) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clients = make(map[string]hermes.Client)
}
