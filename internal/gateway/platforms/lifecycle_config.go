package platforms

import (
	"strconv"
	"strings"
	"time"
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
	if lookup == nil {
		lookup = func(string) string { return "" }
	}
	for _, key := range []string{HermesPlatformPauseAfterFailuresEnv, GormesPlatformPauseAfterFailuresEnv} {
		raw := strings.TrimSpace(lookup(key))
		if raw == "" {
			continue
		}
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			return DefaultPlatformPauseAfterFailures
		}
		return value
	}
	return DefaultPlatformPauseAfterFailures
}

func durationSecondsFromEnv(lookup func(string) string, key string, fallback time.Duration) time.Duration {
	if lookup == nil {
		lookup = func(string) string { return "" }
	}
	raw := strings.TrimSpace(lookup(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	if value < 0 {
		value = 0
	}
	return time.Duration(value * float64(time.Second))
}
