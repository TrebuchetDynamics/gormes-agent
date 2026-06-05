package remoteruntime

import (
	"context"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/tuigateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
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
	raw = textvalue.LowerTrim(raw)
	return strings.HasPrefix(raw, "ws://") || strings.HasPrefix(raw, "wss://")
}
