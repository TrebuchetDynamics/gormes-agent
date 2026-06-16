package gateway

import (
	"os"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/remoteruntime"
)

func ResolveRemoteURL(flagValue string) string {
	if raw := strings.TrimSpace(flagValue); raw != "" {
		return raw
	}
	for _, key := range []string{"GORMES_TUI_GATEWAY_URL", "HERMES_TUI_GATEWAY_URL"} {
		if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
			return raw
		}
	}
	return ""
}

func ResolveRemoteSidecarURL() string {
	for _, key := range []string{"GORMES_TUI_SIDECAR_URL", "HERMES_TUI_SIDECAR_URL"} {
		if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
			return raw
		}
	}
	return ""
}

func IsWebSocketRemoteURL(raw string) bool {
	return remoteruntime.IsWebSocketRemoteURL(raw)
}
