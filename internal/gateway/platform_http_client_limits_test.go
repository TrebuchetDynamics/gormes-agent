package gateway

import (
	"testing"
	"time"
)

func TestPlatformHTTPClientLimits_DefaultsTightenKeepalive(t *testing.T) {
	limits := PlatformHTTPClientLimitsFromEnv(func(string) string { return "" })
	if limits.KeepaliveExpiry <= 0 || limits.KeepaliveExpiry >= 5*time.Second {
		t.Fatalf("KeepaliveExpiry = %s, want positive and below 5s", limits.KeepaliveExpiry)
	}
	if limits.MaxKeepaliveConnections < 1 || limits.MaxKeepaliveConnections > 50 {
		t.Fatalf("MaxKeepaliveConnections = %d, want 1..50", limits.MaxKeepaliveConnections)
	}
}

func TestPlatformHTTPClientLimits_EnvOverridesAndFallbacks(t *testing.T) {
	lookup := func(key string) string {
		switch key {
		case "HERMES_GATEWAY_HTTPX_KEEPALIVE_EXPIRY":
			return "7.5"
		case "HERMES_GATEWAY_HTTPX_MAX_KEEPALIVE":
			return "25"
		default:
			return ""
		}
	}
	limits := PlatformHTTPClientLimitsFromEnv(lookup)
	if limits.KeepaliveExpiry != 7500*time.Millisecond {
		t.Fatalf("KeepaliveExpiry = %s, want 7.5s", limits.KeepaliveExpiry)
	}
	if limits.MaxKeepaliveConnections != 25 {
		t.Fatalf("MaxKeepaliveConnections = %d, want 25", limits.MaxKeepaliveConnections)
	}

	bad := PlatformHTTPClientLimitsFromEnv(func(string) string { return "not-a-number" })
	if bad.KeepaliveExpiry <= 0 || bad.MaxKeepaliveConnections <= 0 {
		t.Fatalf("bad overrides = %+v, want positive defaults", bad)
	}
}
