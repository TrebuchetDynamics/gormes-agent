package main

import (
	"context"
	"net/url"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/network/vpnhost"
)

func TestNavivoxSetupPairingURIIncludesRESTTokenAndURLs(t *testing.T) {
	cfg := config.NavivoxCfg{
		Enabled:      true,
		BindHost:     "127.0.0.1",
		Port:         8765,
		ExposureMode: config.NavivoxExposureLocal,
		AuthMode:     config.NavivoxAuthPairingToken,
		Token:        "setup-secret-token",
	}

	descriptor, err := navivoxSetupPairingURI(cfg)
	if err != nil {
		t.Fatalf("navivoxSetupPairingURI error = %v", err)
	}
	u, err := url.Parse(descriptor)
	if err != nil {
		t.Fatalf("descriptor is not a URI: %v", err)
	}
	if u.Scheme != "navivox" || u.Host != "connect" {
		t.Fatalf("descriptor = %q, want navivox://connect URI", descriptor)
	}
	q := u.Query()
	checks := map[string]string{
		"base_url":      "http://127.0.0.1:8765",
		"websocket_url": "ws://127.0.0.1:8765/v1/navivox/stream",
		"auth_mode":     config.NavivoxAuthPairingToken,
		"rest_token":    cfg.Token,
		"token_required": "true",
	}
	for key, want := range checks {
		if got := q.Get(key); got != want {
			t.Fatalf("descriptor query %s = %q, want %q in %q", key, got, want, descriptor)
		}
	}
}

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

func TestNavivoxSetupBindDefault_WireGuardExposure_PicksDetectedWireGuardIPv4(t *testing.T) {
	prev := vpnhostList
	t.Cleanup(func() { vpnhostList = prev })
	vpnhostList = func(context.Context) ([]vpnhost.Host, error) {
		return []vpnhost.Host{
			{Iface: "tailscale0", Kind: vpnhost.KindTailscale, IPv4: "100.64.1.2"},
			{Iface: "wg0", Kind: vpnhost.KindWireGuard, IPv4: "10.0.0.1"},
		}, nil
	}

	got := navivoxSetupBindDefault(context.Background(), "", config.NavivoxExposureWireGuard)
	if got != "10.0.0.1" {
		t.Fatalf("bind_host = %q, want wireguard IPv4 10.0.0.1", got)
	}
}

func TestNavivoxSetupBindDefault_VPNExposure_PicksFirstDetectedVPNIP(t *testing.T) {
	prev := vpnhostList
	t.Cleanup(func() { vpnhostList = prev })
	vpnhostList = func(context.Context) ([]vpnhost.Host, error) {
		return []vpnhost.Host{
			{Iface: "tun0", Kind: vpnhost.KindTunOther, IPv4: "10.8.0.5"},
			{Iface: "wg0", Kind: vpnhost.KindWireGuard, IPv4: "10.0.0.1"},
		}, nil
	}

	got := navivoxSetupBindDefault(context.Background(), "", config.NavivoxExposureVPN)
	if got != "10.8.0.5" {
		t.Fatalf("bind_host = %q, want first VPN IPv4 10.8.0.5", got)
	}
}

func TestNavivoxSetupBindDefault_CurrentValuePreservedRegardlessOfExposure(t *testing.T) {
	got := navivoxSetupBindDefault(context.Background(), "192.168.5.5", config.NavivoxExposureTailscale)
	if got != "192.168.5.5" {
		t.Fatalf("bind_host = %q, want operator-provided value preserved", got)
	}
}
