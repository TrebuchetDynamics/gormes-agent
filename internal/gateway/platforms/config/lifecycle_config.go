package config

import (
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/platforms/envconfig"
)

const (
	DefaultPlatformConnectTimeout       = 30 * time.Second
	DefaultChannelDisconnectTimeout     = 5 * time.Second
	DefaultPlatformPauseAfterFailures   = 10
	HermesPlatformConnectTimeoutEnv     = "HERMES_GATEWAY_PLATFORM_CONNECT_TIMEOUT"
	HermesChannelDisconnectTimeoutEnv   = "HERMES_GATEWAY_ADAPTER_DISCONNECT_TIMEOUT"
	HermesPlatformPauseAfterFailuresEnv = "HERMES_GATEWAY_PAUSE_AFTER_FAILURES"
	GormesPlatformPauseAfterFailuresEnv = "GORMES_GATEWAY_PAUSE_AFTER_FAILURES"
)

func PlatformConnectTimeoutFromEnv(lookup func(string) string) time.Duration {
	return durationSecondsFromEnv(lookup, HermesPlatformConnectTimeoutEnv, DefaultPlatformConnectTimeout)
}

func ChannelDisconnectTimeoutFromEnv(lookup func(string) string) time.Duration {
	return durationSecondsFromEnv(lookup, HermesChannelDisconnectTimeoutEnv, DefaultChannelDisconnectTimeout)
}

func PlatformPauseThresholdFromEnv(lookup func(string) string) int {
	lookup = envconfig.Lookup(lookup)
	for _, key := range []string{HermesPlatformPauseAfterFailuresEnv, GormesPlatformPauseAfterFailuresEnv} {
		raw := strings.TrimSpace(lookup(key))
		if raw == "" {
			continue
		}
		return envconfig.PositiveInt(raw, DefaultPlatformPauseAfterFailures)
	}
	return DefaultPlatformPauseAfterFailures
}

func durationSecondsFromEnv(lookup func(string) string, key string, fallback time.Duration) time.Duration {
	lookup = envconfig.Lookup(lookup)
	return envconfig.NonNegativeFloatSeconds(lookup(key), fallback)
}
