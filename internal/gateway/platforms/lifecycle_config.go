package platforms

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/platforms/config"
)

const (
	DefaultPlatformConnectTimeout       = config.DefaultPlatformConnectTimeout
	DefaultChannelDisconnectTimeout     = config.DefaultChannelDisconnectTimeout
	DefaultPlatformPauseAfterFailures   = config.DefaultPlatformPauseAfterFailures
	HermesPlatformConnectTimeoutEnv     = config.HermesPlatformConnectTimeoutEnv
	HermesChannelDisconnectTimeoutEnv   = config.HermesChannelDisconnectTimeoutEnv
	HermesPlatformPauseAfterFailuresEnv = config.HermesPlatformPauseAfterFailuresEnv
	GormesPlatformPauseAfterFailuresEnv = config.GormesPlatformPauseAfterFailuresEnv
)

func PlatformConnectTimeoutFromEnv(lookup func(string) string) time.Duration {
	return config.PlatformConnectTimeoutFromEnv(lookup)
}

func ChannelDisconnectTimeoutFromEnv(lookup func(string) string) time.Duration {
	return config.ChannelDisconnectTimeoutFromEnv(lookup)
}

func PlatformPauseThresholdFromEnv(lookup func(string) string) int {
	return config.PlatformPauseThresholdFromEnv(lookup)
}
