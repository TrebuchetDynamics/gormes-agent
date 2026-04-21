package subagent

import "sync"

// SubagentRegistry tracks live subagents so they can be listed or cancelled
// together during shutdown.
type SubagentRegistry interface {
	Register(*Subagent)
	Unregister(id string)
	InterruptAll(message string)
	List() []*Subagent
}

type registry struct {
	mu        sync.RWMutex
	subagents map[string]*Subagent
}

// NewRegistry returns an empty, concurrency-safe SubagentRegistry.
func NewRegistry() SubagentRegistry {
	return &registry{
		subagents: make(map[string]*Subagent),
	}
}

func (r *registry) Register(sa *Subagent) {
	r.mu.Lock()
	r.subagents[sa.ID] = sa
	r.mu.Unlock()
}

func (r *registry) Unregister(id string) {
	r.mu.Lock()
	delete(r.subagents, id)
	r.mu.Unlock()
}

func (r *registry) InterruptAll(message string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, sa := range r.subagents {
		sa.interruptMsg.Store(message)
		sa.cancel()
	}
}

func (r *registry) List() []*Subagent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Subagent, 0, len(r.subagents))
	for _, sa := range r.subagents {
		out = append(out, sa)
	}
	return out
}
