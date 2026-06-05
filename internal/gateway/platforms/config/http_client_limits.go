package config

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/platforms/envconfig"
)

const (
	defaultPlatformKeepaliveExpiry = 2 * time.Second
	defaultPlatformMaxKeepalive    = 10
)

type PlatformHTTPClientLimits struct {
	KeepaliveExpiry         time.Duration
	MaxKeepaliveConnections int
}

func PlatformHTTPClientLimitsFromEnv(lookup func(string) string) PlatformHTTPClientLimits {
	lookup = envconfig.Lookup(lookup)
	return PlatformHTTPClientLimits{
		KeepaliveExpiry:         ParsePlatformKeepaliveExpiry(lookup("HERMES_GATEWAY_HTTPX_KEEPALIVE_EXPIRY")),
		MaxKeepaliveConnections: ParsePlatformMaxKeepalive(lookup("HERMES_GATEWAY_HTTPX_MAX_KEEPALIVE")),
	}
}

func ParsePlatformKeepaliveExpiry(raw string) time.Duration {
	return envconfig.PositiveFloatSeconds(raw, defaultPlatformKeepaliveExpiry)
}

func ParsePlatformMaxKeepalive(raw string) int {
	return envconfig.PositiveInt(raw, defaultPlatformMaxKeepalive)
}
