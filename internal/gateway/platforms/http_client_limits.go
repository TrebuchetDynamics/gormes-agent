package platforms

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/platforms/config"
)

const (
	defaultPlatformKeepaliveExpiry = 2 * time.Second
	defaultPlatformMaxKeepalive    = 10
)

type PlatformHTTPClientLimits = config.PlatformHTTPClientLimits

func PlatformHTTPClientLimitsFromEnv(lookup func(string) string) PlatformHTTPClientLimits {
	return config.PlatformHTTPClientLimitsFromEnv(lookup)
}

func ParsePlatformKeepaliveExpiry(raw string) time.Duration {
	return config.ParsePlatformKeepaliveExpiry(raw)
}

func ParsePlatformMaxKeepalive(raw string) int {
	return config.ParsePlatformMaxKeepalive(raw)
}
