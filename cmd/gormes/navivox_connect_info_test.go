package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/network/vpnhost"
)

func newConnectInfoTestCommand(t *testing.T) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	cmd := &cobra.Command{Use: "test"}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	return cmd, buf
}

func TestNavivoxConnectInfo_Disabled_ReturnsTypedError(t *testing.T) {
	cmd, _ := newConnectInfoTestCommand(t)
	err := runNavivoxConnectInfo(cmd, config.NavivoxCfg{Enabled: false}, false)
	if err == nil {
		t.Fatal("err = nil, want disabled error")
	}
	if !strings.Contains(err.Error(), "[navivox].enabled") {
		t.Errorf("err = %q, want hint about [navivox].enabled", err)
	}
}

func TestNavivoxConnectInfo_LocalMode_PrintsLoopbackOnly_JSON(t *testing.T) {
	prev := vpnhostList
	t.Cleanup(func() { vpnhostList = prev })
	vpnhostList = func(context.Context) ([]vpnhost.Host, error) {
		return []vpnhost.Host{
			{Iface: "tailscale0", Kind: vpnhost.KindTailscale, IPv4: "100.64.1.2"},
		}, nil
	}

	cmd, buf := newConnectInfoTestCommand(t)
	cfg := config.NavivoxCfg{
		Enabled:      true,
		BindHost:     "127.0.0.1",
		Port:         8765,
		ExposureMode: config.NavivoxExposureLocal,
		AuthMode:     config.NavivoxAuthStaticToken,
		Token:        "secret-token-do-not-leak",
	}
	if err := runNavivoxConnectInfo(cmd, cfg, true); err != nil {
		t.Fatal(err)
	}
	var got navivoxConnectInfoReport
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if len(got.Entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1: %+v", len(got.Entries), got.Entries)
	}
	if got.Entries[0].BaseURL != "http://127.0.0.1:8765" {
		t.Errorf("base_url = %q, want http://127.0.0.1:8765", got.Entries[0].BaseURL)
	}
	if got.Entries[0].HostSource != "local" {
		t.Errorf("host_source = %q, want local", got.Entries[0].HostSource)
	}
	if got.Entries[0].HealthzURL != "http://127.0.0.1:8765/healthz" {
		t.Errorf("healthz_url = %q", got.Entries[0].HealthzURL)
	}
	if !got.Entries[0].TokenRequired {
		t.Error("token_required = false, want true for static_token auth")
	}
}

func TestNavivoxConnectInfo_NeverLeaksTokenValue(t *testing.T) {
	prev := vpnhostList
	t.Cleanup(func() { vpnhostList = prev })
	vpnhostList = func(context.Context) ([]vpnhost.Host, error) {
		return []vpnhost.Host{
			{Iface: "tailscale0", Kind: vpnhost.KindTailscale, IPv4: "100.64.1.2"},
		}, nil
	}

	const sensitiveToken = "super-secret-token-9871"
	cfg := config.NavivoxCfg{
		Enabled:      true,
		BindHost:     "100.64.1.2",
		Port:         8765,
		ExposureMode: config.NavivoxExposureTailscale,
		AuthMode:     config.NavivoxAuthStaticToken,
		Token:        sensitiveToken,
	}

	for _, jsonOut := range []bool{true, false} {
		cmd, buf := newConnectInfoTestCommand(t)
		if err := runNavivoxConnectInfo(cmd, cfg, jsonOut); err != nil {
			t.Fatalf("jsonOut=%v: %v", jsonOut, err)
		}
		if strings.Contains(buf.String(), sensitiveToken) {
			t.Errorf("jsonOut=%v: output leaks token value\noutput: %s", jsonOut, buf.String())
		}
	}
}

func TestNavivoxConnectInfo_LayeredAuthRequiresToken(t *testing.T) {
	prev := vpnhostList
	t.Cleanup(func() { vpnhostList = prev })
	vpnhostList = func(context.Context) ([]vpnhost.Host, error) {
		return []vpnhost.Host{{Iface: "tailscale0", Kind: vpnhost.KindTailscale, IPv4: "100.64.1.2"}}, nil
	}

	cmd, buf := newConnectInfoTestCommand(t)
	cfg := config.NavivoxCfg{
		Enabled:      true,
		BindHost:     "100.64.1.2",
		Port:         8765,
		ExposureMode: config.NavivoxExposureTailscale,
		AuthMode:     config.NavivoxAuthTokenAndTailscaleIdentity,
		Token:        "secret-token-do-not-leak",
	}
	if err := runNavivoxConnectInfo(cmd, cfg, true); err != nil {
		t.Fatal(err)
	}
	var got navivoxConnectInfoReport
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if len(got.Entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(got.Entries))
	}
	if !got.Entries[0].TokenRequired {
		t.Error("token_required = false, want true for layered token+identity auth")
	}
	if strings.Contains(buf.String(), cfg.Token) {
		t.Fatalf("connect-info leaked token: %s", buf.String())
	}
}

func TestNavivoxConnectInfo_JSONIncludesWebSocketURLAndBracketsIPv6(t *testing.T) {
	prev := vpnhostList
	t.Cleanup(func() { vpnhostList = prev })
	vpnhostList = func(context.Context) ([]vpnhost.Host, error) {
		return []vpnhost.Host{
			{Iface: "tailscale0", Kind: vpnhost.KindTailscale, IPv6: "fd7a:115c:a1e0::1"},
		}, nil
	}

	cmd, buf := newConnectInfoTestCommand(t)
	cfg := config.NavivoxCfg{
		Enabled:      true,
		BindHost:     "fd7a:115c:a1e0::1",
		Port:         8765,
		ExposureMode: config.NavivoxExposureTailscale,
		AuthMode:     config.NavivoxAuthStaticToken,
		Token:        "x",
	}
	if err := runNavivoxConnectInfo(cmd, cfg, true); err != nil {
		t.Fatal(err)
	}
	var raw struct {
		Entries []map[string]any `json:"entries"`
	}
	if err := json.Unmarshal(buf.Bytes(), &raw); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if len(raw.Entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1: %+v", len(raw.Entries), raw.Entries)
	}
	entry := raw.Entries[0]
	if entry["base_url"] != "http://[fd7a:115c:a1e0::1]:8765" {
		t.Fatalf("base_url = %v, want bracketed IPv6 URL", entry["base_url"])
	}
	if entry["websocket_url"] != "ws://[fd7a:115c:a1e0::1]:8765/v1/navivox/stream" {
		t.Fatalf("websocket_url = %v, want stream URL", entry["websocket_url"])
	}
}

func TestNavivoxConnectInfo_TailscaleMode_JSONShowsVPNEntries(t *testing.T) {
	prev := vpnhostList
	t.Cleanup(func() { vpnhostList = prev })
	vpnhostList = func(context.Context) ([]vpnhost.Host, error) {
		return []vpnhost.Host{
			{Iface: "tailscale0", Kind: vpnhost.KindTailscale, IPv4: "100.64.1.2", IPv6: "fd7a::1"},
		}, nil
	}

	cmd, buf := newConnectInfoTestCommand(t)
	cfg := config.NavivoxCfg{
		Enabled:      true,
		BindHost:     "100.64.1.2",
		Port:         8765,
		ExposureMode: config.NavivoxExposureTailscale,
		AuthMode:     config.NavivoxAuthStaticToken,
		Token:        "x",
	}
	if err := runNavivoxConnectInfo(cmd, cfg, true); err != nil {
		t.Fatal(err)
	}
	var got navivoxConnectInfoReport
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if len(got.Entries) == 0 {
		t.Fatal("len(entries) = 0, want at least one tailscale entry")
	}
	if got.Entries[0].HostSource != "tailscale" {
		t.Errorf("host_source = %q, want tailscale", got.Entries[0].HostSource)
	}
	if got.Entries[0].Host != "100.64.1.2" {
		t.Errorf("host = %q, want 100.64.1.2", got.Entries[0].Host)
	}
	if got.Entries[0].BaseURL != "http://100.64.1.2:8765" {
		t.Errorf("base_url = %q", got.Entries[0].BaseURL)
	}
}

func TestNavivoxConnectInfo_VPNMode_ListsAllDetectedKinds(t *testing.T) {
	prev := vpnhostList
	t.Cleanup(func() { vpnhostList = prev })
	vpnhostList = func(context.Context) ([]vpnhost.Host, error) {
		return []vpnhost.Host{
			{Iface: "tailscale0", Kind: vpnhost.KindTailscale, IPv4: "100.64.1.2"},
			{Iface: "wg0", Kind: vpnhost.KindWireGuard, IPv4: "10.0.0.1"},
			{Iface: "tun0", Kind: vpnhost.KindTunOther, IPv4: "10.8.0.5"},
		}, nil
	}

	cmd, buf := newConnectInfoTestCommand(t)
	cfg := config.NavivoxCfg{
		Enabled:      true,
		BindHost:     "100.64.1.2",
		Port:         8765,
		ExposureMode: config.NavivoxExposureVPN,
		AuthMode:     config.NavivoxAuthStaticToken,
		Token:        "x",
	}
	if err := runNavivoxConnectInfo(cmd, cfg, true); err != nil {
		t.Fatal(err)
	}
	var got navivoxConnectInfoReport
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(got.Entries) != 3 {
		t.Fatalf("len(entries) = %d, want 3 (one per VPN kind): %+v", len(got.Entries), got.Entries)
	}
	wantSources := []string{"tailscale", "wireguard", "tun-other"}
	for i, src := range wantSources {
		if got.Entries[i].HostSource != src {
			t.Errorf("entries[%d].host_source = %q, want %q", i, got.Entries[i].HostSource, src)
		}
	}
}

func TestNavivoxConnectInfo_TextOutputShowsKindLabel(t *testing.T) {
	prev := vpnhostList
	t.Cleanup(func() { vpnhostList = prev })
	vpnhostList = func(context.Context) ([]vpnhost.Host, error) {
		return []vpnhost.Host{
			{Iface: "tailscale0", Kind: vpnhost.KindTailscale, IPv4: "100.64.1.2"},
		}, nil
	}

	cmd, buf := newConnectInfoTestCommand(t)
	cfg := config.NavivoxCfg{
		Enabled:      true,
		BindHost:     "100.64.1.2",
		Port:         8765,
		ExposureMode: config.NavivoxExposureTailscale,
		AuthMode:     config.NavivoxAuthStaticToken,
		Token:        "x",
	}
	if err := runNavivoxConnectInfo(cmd, cfg, false); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	for _, want := range []string{"http://100.64.1.2:8765", "ws://100.64.1.2:8765/v1/navivox/stream", "tailscale", "/healthz"} {
		if !strings.Contains(s, want) {
			t.Errorf("text output missing %q\noutput: %s", want, s)
		}
	}
}
