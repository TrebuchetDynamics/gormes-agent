package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/skip2/go-qrcode"
	"github.com/spf13/cobra"

	channelsmodule "github.com/TrebuchetDynamics/gormes-agent/internal/cli/gormescli/modules/channels"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/network/vpnhost"
)

type navivoxPairOptions struct {
	host           string
	port           int
	noWait         bool
	openNavivox    bool
	noOpenNavivox  bool
	androidPackage string
	printDeeplink  bool
	portExplicit   bool
}

type navivoxPairRuntimeStore interface {
	ReadValidatedRuntimeStatusSnapshot(context.Context) (gateway.RuntimeStatusSnapshot, error)
}

var newNavivoxPairRuntimeStore = func(path string) navivoxPairRuntimeStore {
	return gateway.NewRuntimeStatusStore(path)
}

func newNavivoxPairCommand() *cobra.Command {
	opts := navivoxPairOptions{
		port:           config.NavivoxDefaultPort,
		openNavivox:    defaultOpenNavivoxAndroid(),
		androidPackage: navivoxAndroidPackage,
	}
	cmd := &cobra.Command{
		Use:   "pair",
		Short: "Create a network Navivox pairing handoff",
		Long: `Start a network-reachable Navivox bridge, generate or reuse a pairing token,
write a QR image, print the token for manual entry, open Navivox directly on
Android when available, then wait for the app to connect. The terminal handoff
also prints a compact QR when it fits the current screen.

Use this after the installer recommends Navivox setup:

Keep the Termux session open after Navivox connects; it owns the local bridge.`,
		Example: `  gormes navivox pair
  gormes navivox pair --port 8765
  gormes navivox pair --open-navivox
  gormes navivox pair --no-wait`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts.portExplicit = cmd.Flags().Changed("port")
			return runNavivoxPair(cmd, opts)
		},
	}
	cmd.Flags().StringVar(&opts.host, "host", opts.host, "Navivox bridge host (default: auto-detect network IP; prefers Tailscale)")
	cmd.Flags().IntVar(&opts.port, "port", opts.port, "Navivox bridge port")
	cmd.Flags().BoolVar(&opts.noWait, "no-wait", false, "print the pairing handoff and exit without waiting")
	cmd.Flags().BoolVar(&opts.openNavivox, "open-navivox", opts.openNavivox, "try Android deep-link handoff after the bridge starts")
	cmd.Flags().BoolVar(&opts.noOpenNavivox, "no-open-navivox", false, "do not launch Navivox; keep QR/manual fallback only")
	cmd.Flags().StringVar(&opts.androidPackage, "android-package", opts.androidPackage, "Android package to target for Navivox deep links")
	cmd.Flags().BoolVar(&opts.printDeeplink, "print-deeplink", false, "print navivox://connect descriptor; warning: contains a secret")
	return cmd
}

func runNavivoxPair(cmd *cobra.Command, opts navivoxPairOptions) error {
	target, err := resolveNavivoxPairTarget(cmd.Context(), opts.host)
	if err != nil {
		return err
	}
	if opts.port <= 0 || opts.port > 65535 {
		return fmt.Errorf("navivox pair: invalid port %d", opts.port)
	}
	if err := ensureNoLiveGatewayForNavivoxPair(cmd.Context()); err != nil {
		return err
	}
	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("navivox pair: load config: %w", err)
	}
	token := strings.TrimSpace(cfg.Navivox.Token)
	generatedToken := false
	if token == "" {
		token, err = generateNavivoxSetupToken()
		if err != nil {
			return err
		}
		generatedToken = true
	}
	runtimeCfg := config.NavivoxCfg{
		Enabled:         true,
		BindHost:        target.Host,
		Port:            opts.port,
		ExposureMode:    target.ExposureMode,
		AuthMode:        config.NavivoxAuthPairingToken,
		Token:           token,
		PublicConfirmed: target.PublicConfirmed,
	}
	if err := config.ValidateNavivoxForRuntime(&runtimeCfg); err != nil {
		return err
	}
	bridgeCfg, bridgeStop, bridgeDone, err := startNavivoxPairBridge(cmd.Context(), runtimeCfg, !opts.portExplicit)
	if err != nil {
		return err
	}
	if bridgeCfg.Port != runtimeCfg.Port {
		target.Source += fmt.Sprintf("; port %d busy, using %d", runtimeCfg.Port, bridgeCfg.Port)
	}
	runtimeCfg = bridgeCfg
	if err := persistNavivoxPairConfig(runtimeCfg, token); err != nil {
		_ = stopNavivoxPairBridge(bridgeStop, bridgeDone)
		return err
	}
	baseURL, wsURL := navivoxConnectInfoURLs(runtimeCfg.BindHost, runtimeCfg.Port)
	descriptor := navivoxPairDescriptor(runtimeCfg, baseURL, wsURL)
	qrPath := filepath.Join(config.GormesHome(), "navivox", "pairing.png")
	if err := writeNavivoxPairQR(qrPath, descriptor); err != nil {
		_ = stopNavivoxPairBridge(bridgeStop, bridgeDone)
		return err
	}

	out := cmd.OutOrStdout()
	openAttempted := shouldOpenNavivoxAndroid(opts.openNavivox, opts.noOpenNavivox)
	openSucceeded := false
	openFailure := ""
	if openAttempted {
		if err := openNavivoxAndroid(cmd.Context(), descriptor, opts.androidPackage); err != nil {
			openFailure = err.Error()
		} else {
			openSucceeded = true
		}
	}
	fmt.Fprintln(out, "Navivox pairing ready.")
	fmt.Fprintf(out, "  Bridge: %s\n", baseURL)
	fmt.Fprintf(out, "  Stream: %s\n", wsURL)
	fmt.Fprintf(out, "  Network: %s\n", target.Source)
	fmt.Fprintf(out, "  Token: %s\n", token)
	if generatedToken {
		fmt.Fprintln(out, "  Token source: generated and stored in:")
	} else {
		fmt.Fprintln(out, "  Token source: reused from:")
	}
	fmt.Fprintf(out, "  %s\n", config.EnvPath())
	fmt.Fprintln(out, "  Treat token/QR like WhatsApp Web:")
	fmt.Fprintln(out, "  anyone with it can connect while this bridge is online.")
	terminalQRFallback := false
	if openSucceeded {
		fmt.Fprintln(out, "  Handoff: opened Navivox directly")
	} else if openAttempted {
		fmt.Fprintf(out, "  Handoff: direct open failed (%s); QR fallback saved:\n", openFailure)
		fmt.Fprintf(out, "    %s\n", qrPath)
		terminalQRFallback = true
		fmt.Fprintln(out, "  Secret: QR embeds the network bridge URL and Navivox token.")
	} else {
		fmt.Fprintln(out, "  Handoff: QR fallback saved:")
		fmt.Fprintf(out, "    %s\n", qrPath)
		terminalQRFallback = true
		fmt.Fprintln(out, "  Secret: QR embeds the network bridge URL and Navivox token.")
	}
	if terminalQRFallback {
		if err := renderNavivoxPairTerminalQR(out, runtimeCfg, baseURL, wsURL, qrPath); err != nil {
			return err
		}
	}
	fmt.Fprintln(out, "  Keep this terminal open for this bridge.")
	if opts.printDeeplink {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Deeplink")
		fmt.Fprintln(out, "  Warning: navivox://connect descriptor contains a secret; do not share it.")
		fmt.Fprintf(out, "  %s\n", descriptor)
	}
	if opts.noWait {
		if err := stopNavivoxPairBridge(bridgeStop, bridgeDone); err != nil {
			return err
		}
		fmt.Fprintln(out, "Waiting for Navivox connection skipped (--no-wait).")
		return nil
	}
	fmt.Fprintln(out, "Waiting for Navivox connection... Press Ctrl-C to stop.")
	connected := make(chan error, 1)
	go func() {
		connected <- waitForNavivoxPairConnection(cmd.Context(), baseURL+"/v1/navivox/status", token)
	}()
	for {
		select {
		case <-cmd.Context().Done():
			if err := stopNavivoxPairBridge(bridgeStop, bridgeDone); err != nil {
				return err
			}
			return nil
		case err := <-bridgeDone:
			if err == nil || errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("navivox pair: local bridge stopped: %w", err)
		case err := <-connected:
			connected = nil
			if err != nil {
				if errors.Is(err, context.Canceled) {
					continue
				}
				_ = stopNavivoxPairBridge(bridgeStop, bridgeDone)
				return err
			}
			fmt.Fprintln(out, "Navivox connected. Continue setup in Navivox.")
			fmt.Fprintf(out, "Local bridge remains online: %s\n", baseURL)
			fmt.Fprintln(out, "Keep this Termux session open to keep the local bridge online.")
		}
	}
}

func ensureNoLiveGatewayForNavivoxPair(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	snapshot, err := newNavivoxPairRuntimeStore(config.GatewayRuntimeStatusPath()).ReadValidatedRuntimeStatusSnapshot(ctx)
	if err != nil {
		return fmt.Errorf("navivox pair: runtime status: %w", err)
	}
	if !snapshot.Validation.Live {
		return nil
	}
	pid := snapshot.Validation.PID
	if pid == 0 {
		pid = snapshot.Status.PID
	}
	pidDetail := ""
	if pid > 0 {
		pidDetail = fmt.Sprintf(" pid=%d", pid)
	}
	return fmt.Errorf("navivox pair: live Gormes gateway already running%s; only one gateway can run at a time. Stop it with `gormes gateway stop`, then rerun `gormes navivox pair`", pidDetail)
}

func persistNavivoxPairConfig(runtimeCfg config.NavivoxCfg, token string) error {
	for _, write := range []struct {
		key   string
		value string
	}{
		{"navivox.enabled", "true"},
		{"navivox.bind_host", runtimeCfg.BindHost},
		{"navivox.port", strconv.Itoa(runtimeCfg.Port)},
		{"navivox.exposure_mode", runtimeCfg.ExposureMode},
		{"navivox.auth_mode", runtimeCfg.AuthMode},
		{"navivox.public_confirmed", strconv.FormatBool(runtimeCfg.PublicConfirmed)},
	} {
		if err := config.WriteTOMLValue(config.ConfigPath(), write.key, write.value); err != nil {
			return err
		}
	}
	return config.WriteEnvValue(config.EnvPath(), "GORMES_NAVIVOX_TOKEN", token)
}

type navivoxPairTarget struct {
	Host            string
	ExposureMode    string
	PublicConfirmed bool
	Source          string
}

func resolveNavivoxPairTarget(ctx context.Context, requestedHost string) (navivoxPairTarget, error) {
	requestedHost = strings.TrimSpace(requestedHost)
	if requestedHost != "" {
		return navivoxPairTarget{
			Host:            requestedHost,
			ExposureMode:    navivoxPairExposureForHost(requestedHost),
			PublicConfirmed: !navivoxPairLoopbackHost(requestedHost),
			Source:          "operator override",
		}, nil
	}

	hosts, _ := vpnhostList(ctx)
	for _, h := range hosts {
		if h.Kind != vpnhost.KindTailscale || strings.TrimSpace(h.IPv4) == "" {
			continue
		}
		return navivoxPairTarget{Host: h.IPv4, ExposureMode: config.NavivoxExposureTailscale, Source: "tailscale auto-detected"}, nil
	}
	for _, h := range hosts {
		if strings.TrimSpace(h.IPv4) == "" {
			continue
		}
		switch h.Kind {
		case vpnhost.KindWireGuard:
			return navivoxPairTarget{Host: h.IPv4, ExposureMode: config.NavivoxExposureWireGuard, Source: "wireguard auto-detected"}, nil
		case vpnhost.KindTunOther:
			return navivoxPairTarget{Host: h.IPv4, ExposureMode: config.NavivoxExposureVPN, Source: "vpn auto-detected"}, nil
		}
	}
	if host := navivoxPairLANIPv4(); host != "" {
		return navivoxPairTarget{Host: host, ExposureMode: config.NavivoxExposurePublic, PublicConfirmed: true, Source: "lan auto-detected"}, nil
	}
	return navivoxPairTarget{}, fmt.Errorf("navivox pair: no network IP detected; connect to Tailscale/Wi-Fi or pass --host <network-ip>")
}

func navivoxPairExposureForHost(host string) string {
	if navivoxPairLoopbackHost(host) {
		return config.NavivoxExposureLocal
	}
	return config.NavivoxExposurePublic
}

func navivoxPairLoopbackHost(host string) bool {
	clean := strings.Trim(strings.TrimSpace(host), "[]")
	if clean == "localhost" {
		return true
	}
	if ip := net.ParseIP(clean); ip != nil && ip.IsLoopback() {
		return true
	}
	return false
}

func navivoxPairLANIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		name := strings.ToLower(iface.Name)
		if strings.HasPrefix(name, "docker") || strings.HasPrefix(name, "br-") || strings.HasPrefix(name, "veth") || strings.HasPrefix(name, "virbr") || strings.HasPrefix(name, "podman") {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			ip = ip.To4()
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			return ip.String()
		}
	}
	return ""
}

func startNavivoxPairBridge(ctx context.Context, cfg config.NavivoxCfg, autoPort bool) (config.NavivoxCfg, context.CancelFunc, <-chan error, error) {
	ln, err := net.Listen("tcp", net.JoinHostPort(strings.Trim(cfg.BindHost, "[]"), strconv.Itoa(cfg.Port)))
	if err != nil && autoPort && navivoxPairPortInUse(err) {
		ln, err = net.Listen("tcp", net.JoinHostPort(strings.Trim(cfg.BindHost, "[]"), "0"))
	}
	if err != nil {
		return config.NavivoxCfg{}, nil, nil, fmt.Errorf("navivox pair: start local bridge: %w", err)
	}
	if tcpAddr, ok := ln.Addr().(*net.TCPAddr); ok && tcpAddr.Port > 0 {
		cfg.Port = tcpAddr.Port
	}
	bridgeCtx, stop := context.WithCancel(ctx)
	done := make(chan error, 1)
	inbox := make(chan gateway.InboundEvent, 16)
	handler, err := channelsmodule.NewNavivoxPairBridgeHandler(cfg, inbox)
	if err != nil {
		_ = ln.Close()
		return config.NavivoxCfg{}, nil, nil, err
	}
	server := &http.Server{Handler: handler}
	go func() {
		<-bridgeCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	go func() {
		err := server.Serve(ln)
		if errors.Is(err, http.ErrServerClosed) {
			err = bridgeCtx.Err()
		}
		done <- err
	}()
	return cfg, stop, done, nil
}

func navivoxPairPortInUse(err error) bool {
	return errors.Is(err, syscall.EADDRINUSE) || strings.Contains(strings.ToLower(err.Error()), "address already in use")
}

func waitForNavivoxPairConnection(ctx context.Context, statusURL, token string) error {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
			if err != nil {
				return err
			}
			req.Header.Set("Authorization", "Bearer "+token)
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			var payload struct {
				WSConnections int `json:"ws_connections"`
			}
			decodeErr := json.NewDecoder(resp.Body).Decode(&payload)
			_ = resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("navivox pair: status check returned HTTP %d", resp.StatusCode)
			}
			if decodeErr != nil {
				return fmt.Errorf("navivox pair: decode status: %w", decodeErr)
			}
			if payload.WSConnections > 0 {
				return nil
			}
		}
	}
}

func stopNavivoxPairBridge(stop context.CancelFunc, done <-chan error) error {
	stop()
	select {
	case err := <-done:
		if err == nil || errors.Is(err, context.Canceled) {
			return nil
		}
		return fmt.Errorf("navivox pair: stop local bridge: %w", err)
	case <-time.After(2 * time.Second):
		return fmt.Errorf("navivox pair: local bridge shutdown timed out")
	}
}

func navivoxPairDescriptor(cfg config.NavivoxCfg, baseURL, wsURL string) string {
	values := url.Values{}
	values.Set("base_url", baseURL)
	values.Set("websocket_url", wsURL)
	values.Set("status_url", strings.TrimRight(baseURL, "/")+"/v1/navivox/status")
	values.Set("capabilities_url", strings.TrimRight(baseURL, "/")+"/v1/navivox/capabilities")
	values.Set("setup_handoff", "true")
	values.Set("setup_mutation_policy", "read_only_handoff")
	values.Set("setup_sections", "provider,model,workspace,channels")
	values.Set("setup_entry_screen", "setup.provider")
	values.Set("bridge_keepalive_required", "true")
	values.Set("bridge_lifecycle", "termux_pair_command")
	values.Set("recommended_path", "navivox")
	values.Set("auth_mode", cfg.AuthMode)
	values.Set("exposure_mode", cfg.ExposureMode)
	values.Set("token_required", "true")
	values.Set("rest_token", cfg.Token)
	return (&url.URL{Scheme: "navivox", Host: "connect", RawQuery: values.Encode()}).String()
}

func writeNavivoxPairQR(path, descriptor string) error {
	if strings.TrimSpace(descriptor) == "" {
		return fmt.Errorf("navivox pair: pairing descriptor is empty")
	}
	pngBytes, err := qrcode.Encode(descriptor, qrcode.Medium, 512)
	if err != nil {
		return fmt.Errorf("navivox pair: encode pairing QR: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("navivox pair: create QR directory: %w", err)
	}
	if err := os.WriteFile(path, pngBytes, 0o600); err != nil {
		return fmt.Errorf("navivox pair: write pairing QR: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("navivox pair: secure pairing QR: %w", err)
	}
	return nil
}
