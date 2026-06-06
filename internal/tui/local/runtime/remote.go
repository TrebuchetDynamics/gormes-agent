package runtime

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/local/runtime/gateway"

func ResolveRemoteURL(flagValue string) string {
	return gateway.ResolveRemoteURL(flagValue)
}

func ResolveRemoteSidecarURL() string {
	return gateway.ResolveRemoteSidecarURL()
}

func IsWebSocketRemoteURL(raw string) bool {
	return gateway.IsWebSocketRemoteURL(raw)
}
