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

func reloadableClientContextErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func (c *ReloadableHermesClient) Set(current llm.Client) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.current = current
}

func (c *ReloadableHermesClient) OpenStream(ctx context.Context, req llm.ChatRequest) (llm.Stream, error) {
	if err := reloadableClientContextErr(ctx); err != nil {
		return nil, err
	}
	current := c.get()
	if current == nil {
		return nil, errors.New("gateway_client_unavailable: gateway provider client is not configured")
	}
	return current.OpenStream(ctx, req)
}

func (c *ReloadableHermesClient) OpenRunEvents(ctx context.Context, runID string) (llm.RunEventStream, error) {
	if err := reloadableClientContextErr(ctx); err != nil {
		return nil, err
	}
	current := c.get()
	if current == nil {
		return nil, errors.New("gateway_client_unavailable: gateway provider client is not configured")
	}
	return current.OpenRunEvents(ctx, runID)
}

func (c *ReloadableHermesClient) Health(ctx context.Context) error {
	if err := reloadableClientContextErr(ctx); err != nil {
		return err
	}
	current := c.get()
	if current == nil {
		return errors.New("gateway_client_unavailable: gateway provider client is not configured")
	}
	return current.Health(ctx)
}

func (c *ReloadableHermesClient) ProviderStatus() llm.ProviderStatus {
	return llm.ProviderStatusOf(c.get())
}

func (c *ReloadableHermesClient) get() llm.Client {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.current
}
