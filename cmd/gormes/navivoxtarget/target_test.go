package navivoxtarget

import (
	"context"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/network/vpnhost"
)

func TestResolvePrefersExplicitHost(t *testing.T) {
	target, err := Resolve(context.Background(), "127.0.0.1", func(context.Context) ([]vpnhost.Host, error) {
		t.Fatal("VPN host lister should not be called for explicit host")
		return nil, nil
	})
	if err != nil {
		t.Fatalf("Resolve explicit host: %v", err)
	}
	if target.Host != "127.0.0.1" || target.ExposureMode != config.NavivoxExposureLocal || target.PublicConfirmed || target.Source != "operator override" {
		t.Fatalf("target = %+v", target)
	}
}

func TestResolvePrefersTailscaleNetworkIP(t *testing.T) {
	target, err := Resolve(context.Background(), "", func(context.Context) ([]vpnhost.Host, error) {
		return []vpnhost.Host{
			{Iface: "wg0", Kind: vpnhost.KindWireGuard, IPv4: "10.0.0.4"},
			{Iface: "tailscale0", Kind: vpnhost.KindTailscale, IPv4: "100.64.1.2"},
		}, nil
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if target.Host != "100.64.1.2" || target.ExposureMode != config.NavivoxExposureTailscale || target.Source != "tailscale auto-detected" {
		t.Fatalf("target = %+v, want tailscale network IP", target)
	}
}

func TestLoopbackHostHandlesNamesAndLiterals(t *testing.T) {
	for _, host := range []string{"localhost", "127.0.0.1", "[::1]"} {
		if !LoopbackHost(host) {
			t.Fatalf("LoopbackHost(%q) = false", host)
		}
	}
	if LoopbackHost("100.64.1.2") {
		t.Fatal("LoopbackHost(100.64.1.2) = true")
	}
}
