package setupnavivox

import (
	"net/url"
	"reflect"
	"testing"

	navivoxapp "github.com/TrebuchetDynamics/gormes-agent/internal/app/navivox"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestPairingURIIncludesTokenWhenRequired(t *testing.T) {
	descriptor, err := PairingURI(config.NavivoxCfg{BindHost: "127.0.0.1", Port: 8765, AuthMode: config.NavivoxAuthPairingToken, ExposureMode: config.NavivoxExposureLocal, Token: "nvbx_setup"})
	if err != nil {
		t.Fatalf("PairingURI: %v", err)
	}
	parsed, err := url.Parse(descriptor)
	if err != nil {
		t.Fatalf("parse descriptor: %v", err)
	}
	values := parsed.Query()
	for key, want := range map[string]string{
		"base_url":         "http://127.0.0.1:8765",
		"websocket_url":    "ws://127.0.0.1:8765/v1/navivox/stream",
		"capabilities_url": "http://127.0.0.1:8765/v1/navivox/capabilities",
		"token_required":   "true",
		"rest_token":       "nvbx_setup",
	} {
		if got := values.Get(key); got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestBindDefaultPrefersCurrentThenVPNKind(t *testing.T) {
	hosts := []navivoxapp.VPNHost{
		{Kind: navivoxapp.VPNHostKindWireGuard, IPv4: "10.0.0.2"},
		{Kind: navivoxapp.VPNHostKindTailscale, IPv4: "100.64.1.2"},
	}
	if got := BindDefault("192.168.5.5", config.NavivoxExposureTailscale, hosts); got != "192.168.5.5" {
		t.Fatalf("current BindDefault = %q", got)
	}
	if got := BindDefault("", config.NavivoxExposureTailscale, hosts); got != "100.64.1.2" {
		t.Fatalf("tailscale BindDefault = %q", got)
	}
	if got := BindDefault("", config.NavivoxExposureWireGuard, hosts); got != "10.0.0.2" {
		t.Fatalf("wireguard BindDefault = %q", got)
	}
}

func TestCSVTrimsNonEmptyValues(t *testing.T) {
	got := CSV(" alice@example.com, , bob@example.com ")
	want := []string{"alice@example.com", "bob@example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CSV = %#v, want %#v", got, want)
	}
}
