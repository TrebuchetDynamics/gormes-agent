package main

import (
	"context"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/network/vpnhost"
)

func TestNavivoxSetupBindDefault_TailscaleExposure_PicksDetectedVPNIPv4(t *testing.T) {
	prev := vpnhostList
	t.Cleanup(func() { vpnhostList = prev })
	vpnhostList = func(context.Context) ([]vpnhost.Host, error) {
		return []vpnhost.Host{
			{Iface: "tailscale0", Kind: vpnhost.KindTailscale, IPv4: "100.64.1.2"},
			{Iface: "wg0", Kind: vpnhost.KindWireGuard, IPv4: "10.0.0.1"},
		}, nil
	}

	got := navivoxSetupBindDefault(context.Background(), "", config.NavivoxExposureTailscale)
	if got != "100.64.1.2" {
		t.Fatalf("bind_host = %q, want tailscale IPv4 100.64.1.2 (vpnhost first entry)", got)
	}
}

func TestNavivoxSetupBindDefault_TailscaleExposure_FallsBackToDefaultWhenNoVPN(t *testing.T) {
	prev := vpnhostList
	t.Cleanup(func() { vpnhostList = prev })
	vpnhostList = func(context.Context) ([]vpnhost.Host, error) {
		return nil, nil
	}

	got := navivoxSetupBindDefault(context.Background(), "", config.NavivoxExposureTailscale)
	if got != config.NavivoxDefaultBindHost {
		t.Fatalf("bind_host = %q, want default %q when no VPN detected", got, config.NavivoxDefaultBindHost)
	}
}

func TestNavivoxSetupBindDefault_TailscaleExposure_SkipsTailscaleEntryWithoutIPv4(t *testing.T) {
	prev := vpnhostList
	t.Cleanup(func() { vpnhostList = prev })
	vpnhostList = func(context.Context) ([]vpnhost.Host, error) {
		return []vpnhost.Host{
			{Iface: "tailscale0", Kind: vpnhost.KindTailscale, IPv6: "fd7a:115c:a1e0::1"},
		}, nil
	}

	got := navivoxSetupBindDefault(context.Background(), "", config.NavivoxExposureTailscale)
	if got != config.NavivoxDefaultBindHost {
		t.Fatalf("bind_host = %q, want default fallback when tailscale has no IPv4", got)
	}
}

func TestNavivoxSetupBindDefault_CurrentValuePreservedRegardlessOfExposure(t *testing.T) {
	got := navivoxSetupBindDefault(context.Background(), "192.168.5.5", config.NavivoxExposureTailscale)
	if got != "192.168.5.5" {
		t.Fatalf("bind_host = %q, want operator-provided value preserved", got)
	}
}
