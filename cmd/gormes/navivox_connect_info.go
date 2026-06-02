package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/navivoxconnect"
	"github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/navivoxhandoff"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/network/vpnhost"
)

// navivoxConnectInfoEntry is one connectable Navivox endpoint reachable
// from a specific interface (loopback for local mode, VPN interfaces for
// tailscale/wireguard/vpn modes).
type navivoxConnectInfoEntry struct {
	ServerID        string                              `json:"server_id,omitempty"`
	Host            string                              `json:"host"`
	HostSource      string                              `json:"host_source"`
	Port            int                                 `json:"port"`
	BaseURL         string                              `json:"base_url"`
	HealthzURL      string                              `json:"healthz_url"`
	CapabilitiesURL string                              `json:"capabilities_url"`
	WebSocketURL    string                              `json:"websocket_url"`
	TokenRequired   bool                                `json:"token_required"`
	Transports      []string                            `json:"transports,omitempty"`
	Capabilities    []string                            `json:"capabilities,omitempty"`
	Profiles        []config.NavivoxProfileRoute        `json:"profiles,omitempty"`
	Warnings        []config.NavivoxProfileRouteWarning `json:"warnings,omitempty"`
}

type navivoxConnectInfoReport struct {
	Entries []navivoxConnectInfoEntry `json:"entries"`
}

func newNavivoxCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "navivox",
		Short: "Navivox HTTP channel utilities",
	}
	cmd.AddCommand(newNavivoxPairCommand())
	return cmd
}

type navivoxConnectInfoOptions struct {
	jsonOut        bool
	openNavivox    bool
	noOpenNavivox  bool
	printDeeplink  bool
	androidPackage string
}

func runNavivoxConnectInfo(cmd *cobra.Command, cfg config.NavivoxCfg, jsonOut bool) error {
	return runNavivoxConnectInfoForConfig(cmd, config.Config{Navivox: cfg}, jsonOut)
}

func runNavivoxConnectInfoForConfig(cmd *cobra.Command, cfg config.Config, jsonOut bool) error {
	return runNavivoxConnectInfoForConfigWithOptions(cmd, cfg, navivoxConnectInfoOptions{jsonOut: jsonOut, androidPackage: navivoxAndroidPackage})
}

func runNavivoxConnectInfoForConfigWithOptions(cmd *cobra.Command, cfg config.Config, opts navivoxConnectInfoOptions) error {
	if !cfg.Navivox.Enabled {
		return fmt.Errorf("navivox connect: [navivox].enabled=false; for first-time Android/Termux setup run `gormes navivox pair`; for a persistent gateway run `gormes setup navivox` or set [navivox].enabled=true with a token")
	}
	if opts.jsonOut && opts.printDeeplink {
		return fmt.Errorf("navivox connect: --print-deeplink cannot be combined with --json")
	}
	entries := buildNavivoxConnectInfoEntriesForConfig(cmd, cfg)
	out := cmd.OutOrStdout()
	if shouldOpenNavivoxAndroid(opts.openNavivox, opts.noOpenNavivox) {
		messageOut := out
		if opts.jsonOut {
			messageOut = cmd.ErrOrStderr()
		}
		writeNavivoxConnectOpenStatus(cmd, messageOut, cfg.Navivox, entries, opts.androidPackage)
	}
	if opts.jsonOut {
		return writeNavivoxConnectInfoJSON(out, entries)
	}
	return writeNavivoxConnectInfoTextWithOptions(out, cfg.Navivox, entries, opts)
}

func buildNavivoxConnectInfoEntries(cmd *cobra.Command, cfg config.NavivoxCfg) []navivoxConnectInfoEntry {
	return buildNavivoxConnectInfoEntriesForConfig(cmd, config.Config{Navivox: cfg})
}

func buildNavivoxConnectInfoEntriesForConfig(cmd *cobra.Command, cfg config.Config) []navivoxConnectInfoEntry {
	navCfg := cfg.Navivox
	tokenRequired := navCfg.AuthMode == config.NavivoxAuthStaticToken ||
		navCfg.AuthMode == config.NavivoxAuthPairingToken ||
		navCfg.AuthMode == config.NavivoxAuthTokenAndTailscaleIdentity
	makeEntry := func(host, source string, port int) navivoxConnectInfoEntry {
		base, stream := navivoxConnectInfoURLs(host, port)
		return navivoxConnectInfoEntry{
			Host:            host,
			HostSource:      source,
			Port:            port,
			BaseURL:         base,
			HealthzURL:      base + "/healthz",
			CapabilitiesURL: base + "/v1/navivox/capabilities",
			WebSocketURL:    stream,
			TokenRequired:   tokenRequired,
		}
	}

	if len(navCfg.Servers) > 0 {
		routing := cfg.NavivoxProfileRouting()
		serverRoutes := map[string]config.NavivoxServerRoute{}
		for _, route := range routing.Servers {
			serverRoutes[route.ServerID] = route
		}
		ids := make([]string, 0, len(navCfg.Servers))
		for id := range navCfg.Servers {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		out := make([]navivoxConnectInfoEntry, 0, len(ids))
		for _, id := range ids {
			server := navCfg.Servers[id]
			if !server.Enabled {
				continue
			}
			host, port := navivoxServerBindHostPort(server.Bind, navCfg)
			entry := makeEntry(host, "server:"+id, port)
			entry.ServerID = id
			entry.Transports = append([]string(nil), server.Transports...)
			entry.Capabilities = append([]string(nil), server.Capabilities...)
			if route, ok := serverRoutes[id]; ok {
				entry.Profiles = append([]config.NavivoxProfileRoute(nil), route.Profiles...)
				entry.Warnings = append([]config.NavivoxProfileRouteWarning(nil), route.Warnings...)
			}
			out = append(out, entry)
		}
		if len(out) > 0 {
			return out
		}
	}

	if !config.NavivoxExposureRequiresVPN(navCfg.ExposureMode) {
		return []navivoxConnectInfoEntry{makeEntry(navCfg.BindHost, "local", navCfg.Port)}
	}

	hosts, _ := vpnhostList(cmd.Context())
	out := make([]navivoxConnectInfoEntry, 0, len(hosts))
	for _, h := range hosts {
		if navCfg.ExposureMode == config.NavivoxExposureTailscale && h.Kind != vpnhost.KindTailscale {
			continue
		}
		if navCfg.ExposureMode == config.NavivoxExposureWireGuard && h.Kind != vpnhost.KindWireGuard {
			continue
		}
		ip := h.IPv4
		if ip == "" {
			ip = h.IPv6
		}
		if ip == "" {
			continue
		}
		out = append(out, makeEntry(ip, string(h.Kind), navCfg.Port))
	}
	return out
}

func navivoxServerBindHostPort(bind string, cfg config.NavivoxCfg) (string, int) {
	return navivoxconnect.ServerBindHostPort(bind, cfg.BindHost, cfg.Port)
}

func navivoxConnectInfoURLs(host string, port int) (baseURL, webSocketURL string) {
	return navivoxconnect.URLs(host, port)
}

func writeNavivoxConnectInfoJSON(out io.Writer, entries []navivoxConnectInfoEntry) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(navivoxConnectInfoReport{Entries: entries})
}

func writeNavivoxConnectInfoText(out io.Writer, cfg config.NavivoxCfg, entries []navivoxConnectInfoEntry) error {
	return writeNavivoxConnectInfoTextWithOptions(out, cfg, entries, navivoxConnectInfoOptions{})
}

func writeNavivoxConnectInfoTextWithOptions(out io.Writer, cfg config.NavivoxCfg, entries []navivoxConnectInfoEntry, opts navivoxConnectInfoOptions) error {
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
		fmt.Fprintf(out, "    capabilities: %s\n", e.CapabilitiesURL)
		fmt.Fprintf(out, "    websocket: %s\n", e.WebSocketURL)
		if e.ServerID != "" {
			fmt.Fprintf(out, "    server: %s\n", e.ServerID)
			if len(e.Profiles) > 0 {
				fmt.Fprintf(out, "    profiles: %s\n", navivoxConnectInfoProfileSummary(e.Profiles))
			}
			for _, warning := range e.Warnings {
				fmt.Fprintf(out, "    warning: %s %s\n", warning.Code, warning.ProfileID)
			}
		}
		if opts.printDeeplink {
			descriptor, err := navivoxConnectInfoDescriptor(cfg, e)
			if err != nil {
				return err
			}
			fmt.Fprintln(out, "    Warning: navivox://connect descriptor contains a secret; do not share it.")
			fmt.Fprintf(out, "    %s\n", descriptor)
		}
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "QR pairing:")
	fmt.Fprintln(out, "  Use `gormes navivox pair` for the one-terminal QR pairing flow.")
	fmt.Fprintln(out, "  Use `gormes navivox pair --open-navivox` to hand these URLs directly to Navivox.")
	return nil
}

func writeNavivoxConnectOpenStatus(cmd *cobra.Command, out io.Writer, cfg config.NavivoxCfg, entries []navivoxConnectInfoEntry, androidPackage string) {
	fmt.Fprintln(out, "Opening Navivox directly...")
	if len(entries) == 0 {
		fmt.Fprintln(out, "Could not open Navivox directly: no reachable Navivox interfaces detected")
		fmt.Fprintln(out, "Use the QR image fallback or manual connect-info import.")
		return
	}
	descriptor, err := navivoxConnectInfoDescriptor(cfg, entries[0])
	if err != nil {
		fmt.Fprintf(out, "Could not open Navivox directly: %s\n", redactNavivoxDescriptor(err.Error()))
		fmt.Fprintln(out, "Use the QR image fallback or manual connect-info import.")
		return
	}
	if err := openNavivoxAndroid(cmd.Context(), descriptor, androidPackage); err != nil {
		fmt.Fprintf(out, "Could not open Navivox directly: %s\n", redactNavivoxDescriptor(err.Error()))
		fmt.Fprintln(out, "Use the QR image fallback or manual connect-info import.")
		return
	}
	fmt.Fprintln(out, "If Navivox did not open, use the QR/image fallback or run with --print-deeplink.")
}

func navivoxConnectInfoProfileSummary(profiles []config.NavivoxProfileRoute) string {
	parts := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		label := strings.TrimSpace(profile.DisplayName)
		if label == "" || label == profile.ProfileID {
			parts = append(parts, profile.ProfileID)
			continue
		}
		parts = append(parts, profile.ProfileID+" ("+label+")")
	}
	return strings.Join(parts, ", ")
}

func navivoxConnectInfoDescriptor(cfg config.NavivoxCfg, entry navivoxConnectInfoEntry) (string, error) {
	return navivoxhandoff.ConnectDescriptor(
		entry.BaseURL,
		entry.WebSocketURL,
		entry.CapabilitiesURL,
		cfg.AuthMode,
		cfg.ExposureMode,
		cfg.Token,
		entry.TokenRequired,
	)
}
