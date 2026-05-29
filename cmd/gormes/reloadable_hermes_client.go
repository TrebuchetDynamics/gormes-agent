package main

import (
	"context"
	"errors"
	"sync"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

type reloadableHermesClient struct {
	mu      sync.RWMutex
	current llm.Client
}

var _ llm.Client = (*reloadableHermesClient)(nil)

func newReloadableHermesClient(current llm.Client) *reloadableHermesClient {
	return &reloadableHermesClient{current: current}
}

func (c *reloadableHermesClient) Set(current llm.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.current = current
}

func (c *reloadableHermesClient) OpenStream(ctx context.Context, req llm.ChatRequest) (llm.Stream, error) {
	current := c.get()
	if current == nil {
		return nil, errors.New("gateway_client_unavailable: gateway provider client is not configured")
	}
	return current.OpenStream(ctx, req)
}

func (c *reloadableHermesClient) OpenRunEvents(ctx context.Context, runID string) (llm.RunEventStream, error) {
	current := c.get()
	if current == nil {
		return nil, errors.New("gateway_client_unavailable: gateway provider client is not configured")
	}
	return current.OpenRunEvents(ctx, runID)
}

func (c *reloadableHermesClient) Health(ctx context.Context) error {
	current := c.get()
	if current == nil {
		return errors.New("gateway_client_unavailable: gateway provider client is not configured")
	}
	return current.Health(ctx)
}

func (c *reloadableHermesClient) get() llm.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.current
}
