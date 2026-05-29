package gormescli

import (
	"context"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/tuigateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

type RemoteTUIClient interface {
	Frames() <-chan kernel.RenderFrame
	Submit(context.Context, string) error
	Cancel(context.Context) error
	Close()
}

func DialRemoteTUI(ctx context.Context, remoteURL, sidecarURL string) (RemoteTUIClient, error) {
	if IsWebSocketRemoteURL(remoteURL) {
		return tuigateway.DialWebSocketAttach(ctx, remoteURL, tuigateway.WithSidecarURL(sidecarURL))
	}
	return tuigateway.DialSSE(ctx, remoteURL)
}

func RedactRemoteTUIURL(raw string) string {
	return tuigateway.RedactRemoteURL(raw)
}

func IsWebSocketRemoteURL(raw string) bool {
	raw = strings.ToLower(strings.TrimSpace(raw))
	return strings.HasPrefix(raw, "ws://") || strings.HasPrefix(raw, "wss://")
}
