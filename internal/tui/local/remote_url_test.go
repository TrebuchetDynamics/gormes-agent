package local

import (
	"strings"
	"testing"
)

func TestResolveRemoteURLPrefersFlagThenEnvironment(t *testing.T) {
	t.Setenv("GORMES_TUI_GATEWAY_URL", "ws://gormes-gateway/api/ws")
	t.Setenv("HERMES_TUI_GATEWAY_URL", "ws://hermes-gateway/api/ws")
	if got := ResolveRemoteURL(" http://flag-gateway/events "); got != "http://flag-gateway/events" {
		t.Fatalf("ResolveRemoteURL(flag) = %q; want trimmed flag", got)
	}
	if got := ResolveRemoteURL(""); got != "ws://gormes-gateway/api/ws" {
		t.Fatalf("ResolveRemoteURL(env) = %q; want GORMES_TUI_GATEWAY_URL", got)
	}
}

func TestResolveRemoteURLAcceptsHermesCompatibilityEnvironment(t *testing.T) {
	t.Setenv("GORMES_TUI_GATEWAY_URL", "")
	t.Setenv("HERMES_TUI_GATEWAY_URL", " ws://hermes-gateway/api/ws?token=secret ")
	if got := ResolveRemoteURL(""); got != "ws://hermes-gateway/api/ws?token=secret" {
		t.Fatalf("ResolveRemoteURL(HERMES_TUI_GATEWAY_URL) = %q", got)
	}
}

func TestDoctorTUIStatusReportsRemoteDegradedMode(t *testing.T) {
	got := DoctorStatus().Format()
	lower := strings.ToLower(got)
	for _, want := range []string{"native tui", "go-native bubble tea", "remote", "websocket"} {
		if !strings.Contains(lower, want) {
			t.Errorf("doctor TUI status missing %q:\n%s", want, got)
		}
	}
}
