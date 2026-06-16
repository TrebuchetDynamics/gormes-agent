package config

import (
	"testing"
	"time"
)

func TestPlatformLifecycleTimeoutsFromEnv(t *testing.T) {
	if got := PlatformConnectTimeoutFromEnv(func(string) string { return "" }); got != 30*time.Second {
		t.Fatalf("default connect timeout = %s", got)
	}
	if got := PlatformConnectTimeoutFromEnv(func(string) string { return "1.5" }); got != 1500*time.Millisecond {
		t.Fatalf("connect timeout = %s, want 1.5s", got)
	}
	if got := ChannelDisconnectTimeoutFromEnv(func(string) string { return "-1" }); got != 0 {
		t.Fatalf("negative disconnect timeout = %s, want 0", got)
	}
}

func TestPlatformPauseThresholdFromEnv(t *testing.T) {
	if got := PlatformPauseThresholdFromEnv(func(string) string { return "" }); got != 10 {
		t.Fatalf("default threshold = %d, want 10", got)
	}
	if got := PlatformPauseThresholdFromEnv(func(k string) string {
		if k == HermesPlatformPauseAfterFailuresEnv {
			return "3"
		}
		return ""
	}); got != 3 {
		t.Fatalf("Hermes threshold = %d, want 3", got)
	}
	if got := PlatformPauseThresholdFromEnv(func(k string) string {
		if k == GormesPlatformPauseAfterFailuresEnv {
			return "5"
		}
		return ""
	}); got != 5 {
		t.Fatalf("Gormes threshold = %d, want 5", got)
	}
	if got := PlatformPauseThresholdFromEnv(func(string) string { return "not-a-number" }); got != 10 {
		t.Fatalf("invalid threshold = %d, want 10", got)
	}
}
