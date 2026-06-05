package gateway

import (
	"time"

	gatewayplatforms "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/platforms"
)

type PlatformHTTPClientLimits = gatewayplatforms.PlatformHTTPClientLimits

func PlatformHTTPClientLimitsFromEnv(lookup func(string) string) PlatformHTTPClientLimits {
	return gatewayplatforms.PlatformHTTPClientLimitsFromEnv(lookup)
}

func parsePlatformKeepaliveExpiry(raw string) time.Duration {
	return gatewayplatforms.ParsePlatformKeepaliveExpiry(raw)
}

func parsePlatformMaxKeepalive(raw string) int {
	return gatewayplatforms.ParsePlatformMaxKeepalive(raw)
}
