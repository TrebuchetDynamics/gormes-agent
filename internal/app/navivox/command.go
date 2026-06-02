package navivox

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/network/vpnhost"
)

// ConnectInfoEntry is one connectable Navivox endpoint reachable
// from a specific interface (loopback for local mode, VPN interfaces for
// tailscale/wireguard/vpn modes).
type ConnectInfoEntry struct {
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

type ConnectInfoReport struct {
	Entries []ConnectInfoEntry `json:"entries"`
}

type CommandOptions struct {
	VPNHostList VPNHostLister
}

var vpnhostList VPNHostLister = vpnhost.List

// SetVPNHostList replaces the VPN host listing seam used by Navivox app commands.
func SetVPNHostList(list VPNHostLister) {
	if list == nil {
		list = vpnhost.List
	}
	vpnhostList = list
}

// NewCommand builds the Navivox cobra command tree.
func NewCommand(opts CommandOptions) *cobra.Command {
	if opts.VPNHostList != nil {
		vpnhostList = opts.VPNHostList
	}
	return newNavivoxCommand()
}

func newNavivoxCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "navivox",
		Short: "Navivox HTTP channel utilities",
	}
	cmd.AddCommand(newNavivoxPairCommand())
	return cmd
}

type ConnectInfoOptions struct {
	jsonOut        bool
	openNavivox    bool
	noOpenNavivox  bool
	printDeeplink  bool
	androidPackage string
}

// ConnectInfoOptionsForTest builds connect-info options for command facade tests.
func ConnectInfoOptionsForTest(jsonOut, openNavivox, noOpenNavivox, printDeeplink bool, androidPackage string) ConnectInfoOptions {
	return ConnectInfoOptions{jsonOut: jsonOut, openNavivox: openNavivox, noOpenNavivox: noOpenNavivox, printDeeplink: printDeeplink, androidPackage: androidPackage}
}

func RunConnectInfo(cmd *cobra.Command, cfg config.NavivoxCfg, jsonOut bool) error {
	return runNavivoxConnectInfo(cmd, cfg, jsonOut)
}

func runNavivoxConnectInfo(cmd *cobra.Command, cfg config.NavivoxCfg, jsonOut bool) error {
	return runNavivoxConnectInfoForConfig(cmd, config.Config{Navivox: cfg}, jsonOut)
}

func RunConnectInfoForConfig(cmd *cobra.Command, cfg config.Config, jsonOut bool) error {
	return runNavivoxConnectInfoForConfig(cmd, cfg, jsonOut)
}

func runNavivoxConnectInfoForConfig(cmd *cobra.Command, cfg config.Config, jsonOut bool) error {
	return runNavivoxConnectInfoForConfigWithOptions(cmd, cfg, ConnectInfoOptions{jsonOut: jsonOut, androidPackage: navivoxAndroidPackage})
}

func RunConnectInfoForConfigWithFlags(cmd *cobra.Command, cfg config.Config, jsonOut, openNavivox, noOpenNavivox, printDeeplink bool, androidPackage string) error {
	return runNavivoxConnectInfoForConfigWithOptions(cmd, cfg, ConnectInfoOptions{jsonOut: jsonOut, openNavivox: openNavivox, noOpenNavivox: noOpenNavivox, printDeeplink: printDeeplink, androidPackage: androidPackage})
}

func runNavivoxConnectInfoForConfigWithOptions(cmd *cobra.Command, cfg config.Config, opts ConnectInfoOptions) error {
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

func BuildConnectInfoEntries(cmd *cobra.Command, cfg config.NavivoxCfg) []ConnectInfoEntry {
	return buildNavivoxConnectInfoEntries(cmd, cfg)
}

func buildNavivoxConnectInfoEntries(cmd *cobra.Command, cfg config.NavivoxCfg) []ConnectInfoEntry {
	return buildNavivoxConnectInfoEntriesForConfig(cmd, config.Config{Navivox: cfg})
}

func BuildConnectInfoEntriesForConfig(cmd *cobra.Command, cfg config.Config) []ConnectInfoEntry {
	return buildNavivoxConnectInfoEntriesForConfig(cmd, cfg)
}

func buildNavivoxConnectInfoEntriesForConfig(cmd *cobra.Command, cfg config.Config) []ConnectInfoEntry {
	navCfg := cfg.Navivox
	tokenRequired := navCfg.AuthMode == config.NavivoxAuthStaticToken ||
		navCfg.AuthMode == config.NavivoxAuthPairingToken ||
		navCfg.AuthMode == config.NavivoxAuthTokenAndTailscaleIdentity
	makeEntry := func(host, source string, port int) ConnectInfoEntry {
		base, stream := navivoxConnectInfoURLs(host, port)
		return ConnectInfoEntry{
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
		out := make([]ConnectInfoEntry, 0, len(ids))
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
		return []ConnectInfoEntry{makeEntry(navCfg.BindHost, "local", navCfg.Port)}
	}

	hosts, _ := vpnhostList(cmd.Context())
	out := make([]ConnectInfoEntry, 0, len(hosts))
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

func NavivoxServerBindHostPort(bind string, cfg config.NavivoxCfg) (string, int) {
	return navivoxServerBindHostPort(bind, cfg)
}

func navivoxServerBindHostPort(bind string, cfg config.NavivoxCfg) (string, int) {
	return ServerBindHostPort(bind, cfg.BindHost, cfg.Port)
}

func NavivoxConnectInfoURLs(host string, port int) (baseURL, webSocketURL string) {
	return navivoxConnectInfoURLs(host, port)
}

func navivoxConnectInfoURLs(host string, port int) (baseURL, webSocketURL string) {
	return URLs(host, port)
}

func WriteConnectInfoJSON(out io.Writer, entries []ConnectInfoEntry) error {
	return writeNavivoxConnectInfoJSON(out, entries)
}

func writeNavivoxConnectInfoJSON(out io.Writer, entries []ConnectInfoEntry) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(ConnectInfoReport{Entries: entries})
}

func WriteConnectInfoText(out io.Writer, cfg config.NavivoxCfg, entries []ConnectInfoEntry) error {
	return writeNavivoxConnectInfoText(out, cfg, entries)
}

func writeNavivoxConnectInfoText(out io.Writer, cfg config.NavivoxCfg, entries []ConnectInfoEntry) error {
	return writeNavivoxConnectInfoTextWithOptions(out, cfg, entries, ConnectInfoOptions{})
}

func writeNavivoxConnectInfoTextWithOptions(out io.Writer, cfg config.NavivoxCfg, entries []ConnectInfoEntry, opts ConnectInfoOptions) error {
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

func writeNavivoxConnectOpenStatus(cmd *cobra.Command, out io.Writer, cfg config.NavivoxCfg, entries []ConnectInfoEntry, androidPackage string) {
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

func navivoxConnectInfoDescriptor(cfg config.NavivoxCfg, entry ConnectInfoEntry) (string, error) {
	return ConnectDescriptor(
		entry.BaseURL,
		entry.WebSocketURL,
		entry.CapabilitiesURL,
		cfg.AuthMode,
		cfg.ExposureMode,
		cfg.Token,
		entry.TokenRequired,
	)
}

func RunConnectInfoForConfigWithOptions(cmd *cobra.Command, cfg config.Config, opts ConnectInfoOptions) error {
	return runNavivoxConnectInfoForConfigWithOptions(cmd, cfg, opts)
}

func ConnectInfoDescriptor(cfg config.NavivoxCfg, entry ConnectInfoEntry) (string, error) {
	return navivoxConnectInfoDescriptor(cfg, entry)
}
