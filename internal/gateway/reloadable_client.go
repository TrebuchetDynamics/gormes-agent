package gateway

import (
	"context"
	"errors"
	"sync"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

type ReloadableHermesClient struct {
	mu      sync.RWMutex
	current llm.Client
}

var _ llm.Client = (*ReloadableHermesClient)(nil)

func NewReloadableHermesClient(current llm.Client) *ReloadableHermesClient {
	return &ReloadableHermesClient{current: current}
}

func (c *ReloadableHermesClient) Set(current llm.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.current = current
}

func (c *ReloadableHermesClient) OpenStream(ctx context.Context, req llm.ChatRequest) (llm.Stream, error) {
	current := c.get()
	if current == nil {
		return nil, errors.New("gateway_client_unavailable: gateway provider client is not configured")
	}
	return current.OpenStream(ctx, req)
}

func (c *ReloadableHermesClient) OpenRunEvents(ctx context.Context, runID string) (llm.RunEventStream, error) {
	current := c.get()
	if current == nil {
		return nil, errors.New("gateway_client_unavailable: gateway provider client is not configured")
	}
	return current.OpenRunEvents(ctx, runID)
}

func (c *ReloadableHermesClient) Health(ctx context.Context) error {
	current := c.get()
	if current == nil {
		return errors.New("gateway_client_unavailable: gateway provider client is not configured")
	}
	return current.Health(ctx)
}

func (c *ReloadableHermesClient) get() llm.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.current
}
