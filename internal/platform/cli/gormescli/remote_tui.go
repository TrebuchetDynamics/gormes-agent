package gormescli

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/remoteruntime"
)

type RemoteTUIClient = remoteruntime.RemoteTUIClient

func DialRemoteTUI(ctx context.Context, remoteURL, sidecarURL string) (RemoteTUIClient, error) {
	return remoteruntime.DialRemoteTUI(ctx, remoteURL, sidecarURL)
}

func RedactRemoteTUIURL(raw string) string {
	return remoteruntime.RedactRemoteTUIURL(raw)
}

func IsWebSocketRemoteURL(raw string) bool {
	return remoteruntime.IsWebSocketRemoteURL(raw)
}
