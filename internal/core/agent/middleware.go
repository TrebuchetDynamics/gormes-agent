package agent

import "strings"

// Middleware processes an agent turn with Before/After lifecycle hooks.
// Before runs in chain order (first → last) before the turn starts.
// After runs in reverse chain order (last → first) after the turn ends.
// A Before error aborts the chain immediately; subsequent Before hooks
// and the turn itself are skipped, but After hooks for already-executed
// middlewares still run.
type Middleware interface {
	Name() string
	Before(ctx *MiddlewareContext) error
	After(ctx *MiddlewareContext) error
}

// MiddlewareContext carries per-turn state through the middleware chain.
type MiddlewareContext struct {
	ThreadID  string
	SessionID string
	Model     string
	Data      map[string]any
}

// MiddlewareChain is an ordered, inspectable list of middlewares.
type MiddlewareChain struct {
	middlewares []Middleware
}

func NewMiddlewareChain(middlewares ...Middleware) *MiddlewareChain {
	chain := &MiddlewareChain{}
	chain.middlewares = append(chain.middlewares, middlewares...)
	return chain
}

func (c *MiddlewareChain) Add(m Middleware) {
	c.middlewares = append(c.middlewares, m)
}

func (c *MiddlewareChain) Names() []string {
	names := make([]string, len(c.middlewares))
	for i, m := range c.middlewares {
		names[i] = m.Name()
	}
	return names
}

func (c *MiddlewareChain) Dump() string {
	return "middleware chain: [" + strings.Join(c.Names(), " ") + "]"
}

func (c *MiddlewareChain) Before(ctx *MiddlewareContext) error {
	for _, m := range c.middlewares {
		if err := m.Before(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (c *MiddlewareChain) After(ctx *MiddlewareContext) error {
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		if err := c.middlewares[i].After(ctx); err != nil {
			return err
		}
	}
	return nil
}
