package navivox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	channelnavivox "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/navivox"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

type PairOptions struct {
	Host           string
	Port           int
	NoWait         bool
	OpenNavivox    bool
	NoOpenNavivox  bool
	AndroidPackage string
	PrintDeeplink  bool
	PortExplicit   bool
}

type PairRuntimeStore interface {
	ReadValidatedRuntimeStatusSnapshot(context.Context) (gateway.RuntimeStatusSnapshot, error)
}

type navivoxPairRuntimeStore = PairRuntimeStore

var newNavivoxPairRuntimeStore = func(path string) PairRuntimeStore {
	return gateway.NewRuntimeStatusStore(path)
}

func SetPairRuntimeStoreFactory(factory func(string) PairRuntimeStore) func(string) PairRuntimeStore {
	previous := newNavivoxPairRuntimeStore
	if factory == nil {
		newNavivoxPairRuntimeStore = func(path string) PairRuntimeStore { return gateway.NewRuntimeStatusStore(path) }
	} else {
		newNavivoxPairRuntimeStore = factory
	}
	return previous
}

func NewPairCommand() *cobra.Command {
	return newNavivoxPairCommand()
}

func newNavivoxPairCommand() *cobra.Command {
	opts := PairOptions{
		Port:           config.NavivoxDefaultPort,
		OpenNavivox:    defaultOpenNavivoxAndroid(),
		AndroidPackage: navivoxAndroidPackage,
	}
	cmd := &cobra.Command{
		Use:   "pair",
		Short: "Create a network Navivox pairing handoff",
		Long: `Start a network-reachable Navivox bridge, generate a temporary one-device pairing token,
write a QR image, print the token for manual entry, open Navivox directly on
Android when available, then wait for the app to connect. The terminal handoff
also prints a compact QR when it fits the current screen.

The token is not stored in Gormes config; it expires when this command stops.

Use this after the installer recommends Navivox setup:

Keep the Termux session open after Navivox connects; it owns the local bridge.`,
		Example: `  gormes navivox pair
  gormes navivox pair --port 8765
  gormes navivox pair --open-navivox
  gormes navivox pair --no-wait`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts.PortExplicit = cmd.Flags().Changed("port")
			return runNavivoxPair(cmd, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Host, "host", opts.Host, "Navivox bridge host (default: auto-detect network IP; prefers Tailscale)")
	cmd.Flags().IntVar(&opts.Port, "port", opts.Port, "Navivox bridge port")
	cmd.Flags().BoolVar(&opts.NoWait, "no-wait", false, "print the pairing handoff and exit without waiting")
	cmd.Flags().BoolVar(&opts.OpenNavivox, "open-navivox", opts.OpenNavivox, "try Android deep-link handoff after the bridge starts")
	cmd.Flags().BoolVar(&opts.NoOpenNavivox, "no-open-navivox", false, "do not launch Navivox; keep QR/manual fallback only")
	cmd.Flags().StringVar(&opts.AndroidPackage, "android-package", opts.AndroidPackage, "Android package to target for Navivox deep links")
	cmd.Flags().BoolVar(&opts.PrintDeeplink, "print-deeplink", false, "print navivox://connect descriptor; warning: contains a secret")
	return cmd
}

func RunPair(cmd *cobra.Command, opts PairOptions) error {
	return runNavivoxPair(cmd, opts)
}

func runNavivoxPair(cmd *cobra.Command, opts PairOptions) error {
	target, err := resolveNavivoxPairTarget(cmd.Context(), opts.Host)
	if err != nil {
		return err
	}
	if opts.Port <= 0 || opts.Port > 65535 {
		return fmt.Errorf("navivox pair: invalid port %d", opts.Port)
	}
	token, err := generateNavivoxSetupToken()
	if err != nil {
		return err
	}
	runtimeCfg := config.NavivoxCfg{
		Enabled:         true,
		BindHost:        target.Host,
		Port:            opts.Port,
		ExposureMode:    target.ExposureMode,
		AuthMode:        config.NavivoxAuthPairingToken,
		Token:           token,
		PublicConfirmed: target.PublicConfirmed,
	}
	if opts.PortExplicit {
		if err := ensureNoLiveGatewayForNavivoxPair(cmd.Context()); err != nil {
			return err
		}
	}
	if err := config.ValidateNavivoxForRuntime(&runtimeCfg); err != nil {
		return err
	}
	requestedPort := runtimeCfg.Port
	bridgeCfg, bridgeStop, bridgeDone, err := startNavivoxPairBridge(cmd.Context(), runtimeCfg, !opts.PortExplicit)
	if err != nil {
		return err
	}
	portChanged := bridgeCfg.Port != requestedPort
	runtimeCfg = bridgeCfg
	baseURL, wsURL := navivoxConnectInfoURLs(runtimeCfg.BindHost, runtimeCfg.Port)
	descriptor := navivoxPairDescriptor(runtimeCfg, baseURL, wsURL)
	qrPath := filepath.Join(config.GormesHome(), "cache", "navivox", "pairing.png")
	if err := writeNavivoxPairQR(qrPath, descriptor); err != nil {
		_ = stopNavivoxPairBridge(bridgeStop, bridgeDone)
		return err
	}

	out := cmd.OutOrStdout()
	openAttempted := shouldOpenNavivoxAndroid(opts.OpenNavivox, opts.NoOpenNavivox)
	openSucceeded := false
	openFailed := false
	if openAttempted {
		if err := openNavivoxAndroid(cmd.Context(), descriptor, opts.AndroidPackage); err != nil {
			openFailed = true
		} else {
			openSucceeded = true
		}
	}
	fmt.Fprintln(out, "Navivox pairing ready.")
	fmt.Fprintf(out, "  URL: %s\n", baseURL)
	fmt.Fprintf(out, "  Token: %s\n", token)
	if portChanged {
		fmt.Fprintf(out, "  Port: %d busy; using %d\n", requestedPort, runtimeCfg.Port)
	}
	fmt.Fprintln(out, "  Keep token/QR private.")
	fmt.Fprintln(out, "  Temporary: one device; expires when this command stops.")
	if openSucceeded {
		fmt.Fprintln(out, "  Opened Navivox.")
	} else {
		if openFailed {
			fmt.Fprintln(out, "  Open failed; use QR.")
		}
		fmt.Fprintf(out, "  QR: %s\n", qrPath)
		if err := renderNavivoxPairTerminalQR(out, runtimeCfg, baseURL, wsURL, qrPath); err != nil {
			return err
		}
	}
	fmt.Fprintln(out, "  Keep this terminal open.")

	if opts.PrintDeeplink {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Deeplink")
		fmt.Fprintln(out, "  Warning: navivox://connect descriptor contains a secret; do not share it.")
		fmt.Fprintf(out, "  %s\n", descriptor)
	}
	if opts.NoWait {
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
			fmt.Fprintln(out, "Navivox connected.")
			fmt.Fprintf(out, "Bridge stays online: %s\n", baseURL)
			fmt.Fprintln(out, "Keep this terminal open.")
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
	return nil
}

type navivoxPairTarget = Target

func resolveNavivoxPairTarget(ctx context.Context, requestedHost string) (navivoxPairTarget, error) {
	return Resolve(ctx, requestedHost, vpnhostList)
}

func navivoxPairExposureForHost(host string) string {
	return ExposureForHost(host)
}

func navivoxPairLoopbackHost(host string) bool {
	return LoopbackHost(host)
}

func navivoxPairLANIPv4() string {
	return LANIPv4()
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
	channel, err := channelnavivox.NewChannel(cfg, nil, channelnavivox.WithSingleUsePairingStream())
	if err != nil {
		stop()
		_ = ln.Close()
		return config.NavivoxCfg{}, nil, nil, fmt.Errorf("navivox pair: create local bridge: %w", err)
	}
	server := &http.Server{Handler: channel.Handler(inbox)}
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

func PairDescriptorForConfig(cfg config.NavivoxCfg, baseURL, wsURL string) string {
	return navivoxPairDescriptor(cfg, baseURL, wsURL)
}

func navivoxPairDescriptor(cfg config.NavivoxCfg, baseURL, wsURL string) string {
	return PairDescriptor(cfg.AuthMode, cfg.ExposureMode, cfg.Token, baseURL, wsURL)
}

func WritePairQR(path, descriptor string) error { return writeNavivoxPairQR(path, descriptor) }

func writeNavivoxPairQR(path, descriptor string) error {
	return WritePNG(path, descriptor)
}

func EnsureNoLiveGatewayForPair(ctx context.Context) error {
	return ensureNoLiveGatewayForNavivoxPair(ctx)
}

func ResolvePairTarget(ctx context.Context, requestedHost string) (Target, error) {
	return resolveNavivoxPairTarget(ctx, requestedHost)
}

func StartPairBridge(ctx context.Context, cfg config.NavivoxCfg, autoPort bool) (config.NavivoxCfg, context.CancelFunc, <-chan error, error) {
	return startNavivoxPairBridge(ctx, cfg, autoPort)
}

func WaitForPairConnection(ctx context.Context, statusURL, token string) error {
	return waitForNavivoxPairConnection(ctx, statusURL, token)
}

func StopPairBridge(stop context.CancelFunc, done <-chan error) error {
	return stopNavivoxPairBridge(stop, done)
}

func PairPortInUse(err error) bool { return navivoxPairPortInUse(err) }
