package main

import (
	"encoding/json"
	"fmt"
	"io"

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
	cmd.AddCommand(newNavivoxConnectInfoCommand())
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
	return writeNavivoxConnectInfoText(out, entries)
}

func buildNavivoxConnectInfoEntries(cmd *cobra.Command, cfg config.NavivoxCfg) []navivoxConnectInfoEntry {
	tokenRequired := cfg.AuthMode == config.NavivoxAuthStaticToken ||
		cfg.AuthMode == config.NavivoxAuthPairingToken
	makeEntry := func(host, source string) navivoxConnectInfoEntry {
		base := fmt.Sprintf("http://%s:%d", host, cfg.Port)
		return navivoxConnectInfoEntry{
			Host:          host,
			HostSource:    source,
			Port:          cfg.Port,
			BaseURL:       base,
			HealthzURL:    base + "/healthz",
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

func writeNavivoxConnectInfoJSON(out io.Writer, entries []navivoxConnectInfoEntry) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(navivoxConnectInfoReport{Entries: entries})
}

func writeNavivoxConnectInfoText(out io.Writer, entries []navivoxConnectInfoEntry) error {
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
	}
	return nil
}
