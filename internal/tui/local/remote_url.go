package local

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/local/runtime"

func ResolveRemoteURL(flagValue string) string {
	return runtime.ResolveRemoteURL(flagValue)
}

func ResolveRemoteSidecarURL() string {
	return runtime.ResolveRemoteSidecarURL()
}

func IsWebSocketRemoteURL(raw string) bool {
	return runtime.IsWebSocketRemoteURL(raw)
}
