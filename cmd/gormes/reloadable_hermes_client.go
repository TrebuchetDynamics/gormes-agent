package main

import (
	"context"
	"errors"
	"sync"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

type reloadableHermesClient struct {
	mu      sync.RWMutex
	current hermes.Client
}

var _ hermes.Client = (*reloadableHermesClient)(nil)

func newReloadableHermesClient(current hermes.Client) *reloadableHermesClient {
	return &reloadableHermesClient{current: current}
}

func (c *reloadableHermesClient) Set(current hermes.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.current = current
}

func (c *reloadableHermesClient) OpenStream(ctx context.Context, req hermes.ChatRequest) (hermes.Stream, error) {
	current := c.get()
	if current == nil {
		return nil, errors.New("gateway_client_unavailable: gateway provider client is not configured")
	}
	return current.OpenStream(ctx, req)
}

func (c *reloadableHermesClient) OpenRunEvents(ctx context.Context, runID string) (hermes.RunEventStream, error) {
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

func (c *reloadableHermesClient) get() hermes.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.current
}
