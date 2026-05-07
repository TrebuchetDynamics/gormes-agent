package gateway

import (
	"strconv"
	"strings"
	"time"
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
	if lookup == nil {
		lookup = func(string) string { return "" }
	}
	return PlatformHTTPClientLimits{
		KeepaliveExpiry:         parsePlatformKeepaliveExpiry(lookup("HERMES_GATEWAY_HTTPX_KEEPALIVE_EXPIRY")),
		MaxKeepaliveConnections: parsePlatformMaxKeepalive(lookup("HERMES_GATEWAY_HTTPX_MAX_KEEPALIVE")),
	}
}

func parsePlatformKeepaliveExpiry(raw string) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultPlatformKeepaliveExpiry
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value <= 0 {
		return defaultPlatformKeepaliveExpiry
	}
	return time.Duration(value * float64(time.Second))
}

func parsePlatformMaxKeepalive(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultPlatformMaxKeepalive
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return defaultPlatformMaxKeepalive
	}
	return value
}
