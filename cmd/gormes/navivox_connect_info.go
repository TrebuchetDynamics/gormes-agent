package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/skip2/go-qrcode"
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/network/vpnhost"
)

// navivoxConnectInfoEntry is one connectable Navivox endpoint reachable
// from a specific interface (loopback for local mode, VPN interfaces for
// tailscale/wireguard/vpn modes).
type navivoxConnectInfoEntry struct {
	Host          string `json:"host"`
	HostSource    string `json:"host_source"`
	Port          int    `json:"port"`
	BaseURL       string `json:"base_url"`
	HealthzURL    string `json:"healthz_url"`
	WebSocketURL  string `json:"websocket_url"`
	TokenRequired bool   `json:"token_required"`
}

type navivoxConnectInfoReport struct {
	Entries []navivoxConnectInfoEntry `json:"entries"`
}

func newNavivoxCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "navivox",
		Short: "Navivox HTTP channel utilities",
	}
	cmd.AddCommand(newNavivoxConnectInfoCommand(), newNavivoxPairCommand())
	return cmd
}

func newNavivoxConnectInfoCommand() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "connect-info",
		Short: "Print Navivox connect URLs for active VPN/local interfaces",
		Long: `Print one connect URL per interface where the running Navivox HTTP channel
should be reachable. Loopback is shown only for exposure_mode=local; VPN-class
modes (tailscale, wireguard, vpn) list every active VPN interface detected by
internal/network/vpnhost. The static_token value is never printed; only a
token_required flag.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(nil)
			if err != nil {
				return err
			}
			return runNavivoxConnectInfo(cmd, cfg.Navivox, jsonOut)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	return cmd
}

func runNavivoxConnectInfo(cmd *cobra.Command, cfg config.NavivoxCfg, jsonOut bool) error {
	if !cfg.Enabled {
		return fmt.Errorf("navivox connect-info: [navivox].enabled=false; set [navivox].enabled=true in config.toml")
	}
	entries := buildNavivoxConnectInfoEntries(cmd, cfg)
	out := cmd.OutOrStdout()
	if jsonOut {
		return writeNavivoxConnectInfoJSON(out, entries)
	}
	return writeNavivoxConnectInfoText(out, cfg, entries)
}

func buildNavivoxConnectInfoEntries(cmd *cobra.Command, cfg config.NavivoxCfg) []navivoxConnectInfoEntry {
	tokenRequired := cfg.AuthMode == config.NavivoxAuthStaticToken ||
		cfg.AuthMode == config.NavivoxAuthPairingToken ||
		cfg.AuthMode == config.NavivoxAuthTokenAndTailscaleIdentity
	makeEntry := func(host, source string) navivoxConnectInfoEntry {
		base, stream := navivoxConnectInfoURLs(host, cfg.Port)
		return navivoxConnectInfoEntry{
			Host:          host,
			HostSource:    source,
			Port:          cfg.Port,
			BaseURL:       base,
			HealthzURL:    base + "/healthz",
			WebSocketURL:  stream,
			TokenRequired: tokenRequired,
		}
	}

	if !config.NavivoxExposureRequiresVPN(cfg.ExposureMode) {
		return []navivoxConnectInfoEntry{makeEntry(cfg.BindHost, "local")}
	}

	hosts, _ := vpnhostList(cmd.Context())
	out := make([]navivoxConnectInfoEntry, 0, len(hosts))
	for _, h := range hosts {
		if cfg.ExposureMode == config.NavivoxExposureTailscale && h.Kind != vpnhost.KindTailscale {
			continue
		}
		if cfg.ExposureMode == config.NavivoxExposureWireGuard && h.Kind != vpnhost.KindWireGuard {
			continue
		}
		ip := h.IPv4
		if ip == "" {
			ip = h.IPv6
		}
		if ip == "" {
			continue
		}
		out = append(out, makeEntry(ip, string(h.Kind)))
	}
	return out
}

func navivoxConnectInfoURLs(host string, port int) (baseURL, webSocketURL string) {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	hostPort := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	baseURL = "http://" + hostPort
	webSocketURL = "ws://" + hostPort + "/v1/navivox/stream"
	return baseURL, webSocketURL
}

func writeNavivoxConnectInfoJSON(out io.Writer, entries []navivoxConnectInfoEntry) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(navivoxConnectInfoReport{Entries: entries})
}

func writeNavivoxConnectInfoText(out io.Writer, cfg config.NavivoxCfg, entries []navivoxConnectInfoEntry) error {
	if len(entries) == 0 {
		fmt.Fprintln(out, "No reachable Navivox interfaces detected. If exposure_mode requires VPN, ensure Tailscale or WireGuard is up.")
		return nil
	}
	fmt.Fprintln(out, "Navivox connect URLs:")
	for _, e := range entries {
		tokenNote := ""
		if e.TokenRequired {
			tokenNote = " (token required)"
		}
		fmt.Fprintf(out, "  - %s  (%s)%s\n", e.BaseURL, e.HostSource, tokenNote)
		fmt.Fprintf(out, "    healthz: %s\n", e.HealthzURL)
		fmt.Fprintf(out, "    websocket: %s\n", e.WebSocketURL)
		qr, err := navivoxConnectInfoTerminalQR(cfg, e)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, "    Scan this QR from Navivox:")
		for _, line := range strings.Split(strings.TrimRight(qr, "\n"), "\n") {
			fmt.Fprintf(out, "    %s\n", line)
		}
		fmt.Fprintln(out, "    navivox://connect descriptor is encoded in the QR.")
		fmt.Fprintln(out, "    QR payload includes the token when required; the raw token is not printed.")
	}
	return nil
}

func navivoxConnectInfoTerminalQR(cfg config.NavivoxCfg, entry navivoxConnectInfoEntry) (string, error) {
	descriptor, err := navivoxConnectInfoDescriptor(cfg, entry)
	if err != nil {
		return "", err
	}
	qr, err := qrcode.New(descriptor, qrcode.Medium)
	if err != nil {
		return "", fmt.Errorf("navivox connect-info: encode terminal QR: %w", err)
	}
	return qr.ToSmallString(false), nil
}

func navivoxConnectInfoDescriptor(cfg config.NavivoxCfg, entry navivoxConnectInfoEntry) (string, error) {
	values := url.Values{}
	values.Set("base_url", entry.BaseURL)
	values.Set("websocket_url", entry.WebSocketURL)
	values.Set("auth_mode", strings.TrimSpace(cfg.AuthMode))
	values.Set("exposure_mode", strings.TrimSpace(cfg.ExposureMode))
	values.Set("token_required", strconv.FormatBool(entry.TokenRequired))
	if entry.TokenRequired {
		token := strings.TrimSpace(cfg.Token)
		if token == "" {
			return "", fmt.Errorf("navivox connect-info: token auth selected but token is empty")
		}
		values.Set("rest_token", token)
	}
	return (&url.URL{Scheme: "navivox", Host: "connect", RawQuery: values.Encode()}).String(), nil
}
