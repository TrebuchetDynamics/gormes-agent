package gateway

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/proxy"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

// ErrProxyBusy is returned when proxy mode is asked to start a second turn
// before the current remote stream has reached a terminal frame.
var ErrProxyBusy = proxy.ErrProxyBusy

// ProxySubmitterConfig wires gateway proxy mode to an OpenAI-compatible
// Gormes API server.
type ProxySubmitterConfig struct {
	BaseURL       string
	APIKey        string
	Model         string
	History       []llm.Message
	Client        llm.Client
	RuntimeStatus RuntimeStatusWriter
}

// ProxySubmitter satisfies the gateway manager's kernel submitter contract by
// forwarding each turn to a remote /v1/chat/completions stream.
type ProxySubmitter = proxy.ProxySubmitter

// NewProxySubmitter constructs a proxy-mode submitter. The default client uses
// the same HTTP+SSE implementation as the native kernel.
func NewProxySubmitter(cfg ProxySubmitterConfig) (*ProxySubmitter, error) {
	return proxy.NewProxySubmitter(proxy.ProxySubmitterConfig{
		BaseURL:       cfg.BaseURL,
		APIKey:        cfg.APIKey,
		Model:         cfg.Model,
		History:       cfg.History,
		Client:        cfg.Client,
		RuntimeStatus: proxyRuntimeStatusWriter{writer: cfg.RuntimeStatus},
	})
}

type proxyRuntimeStatusWriter struct {
	writer RuntimeStatusWriter
}

func (w proxyRuntimeStatusWriter) UpdateRuntimeStatus(ctx context.Context, update proxy.RuntimeStatusUpdate) error {
	if w.writer == nil {
		return nil
	}
	return w.writer.UpdateRuntimeStatus(ctx, RuntimeStatusUpdate{
		ProxyState:        update.ProxyState,
		ProxyURL:          update.ProxyURL,
		ProxyErrorMessage: update.ProxyErrorMessage,
	})
}
