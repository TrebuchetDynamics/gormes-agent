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
	"time"

	"github.com/skip2/go-qrcode"
	"github.com/spf13/cobra"

	navivoxchannel "github.com/TrebuchetDynamics/gormes-agent/internal/channels/navivox"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

type navivoxPairOptions struct {
	host   string
	port   int
	noWait bool
}

func newNavivoxPairCommand() *cobra.Command {
	opts := navivoxPairOptions{
		host: config.NavivoxDefaultBindHost,
		port: config.NavivoxDefaultPort,
	}
	cmd := &cobra.Command{
		Use:   "pair",
		Short: "Create a local Navivox pairing handoff",
		Long: `Start a local Navivox bridge, generate a pairing token, write a QR image,
print the localhost URL, then wait for the Android app to connect.

Use this after the installer recommends Navivox setup:

Keep the Termux session open after Navivox connects; it owns the local bridge.`,
		Example: `  gormes navivox pair
  gormes navivox pair --port 8765
  gormes navivox pair --no-wait`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runNavivoxPair(cmd, opts)
		},
	}
	cmd.Flags().StringVar(&opts.host, "host", opts.host, "local Navivox bridge host")
	cmd.Flags().IntVar(&opts.port, "port", opts.port, "local Navivox bridge port")
	cmd.Flags().BoolVar(&opts.noWait, "no-wait", false, "print the pairing handoff and exit without waiting")
	return cmd
}

func runNavivoxPair(cmd *cobra.Command, opts navivoxPairOptions) error {
	host := strings.TrimSpace(opts.host)
	if host == "" {
		host = config.NavivoxDefaultBindHost
	}
	if opts.port <= 0 || opts.port > 65535 {
		return fmt.Errorf("navivox pair: invalid port %d", opts.port)
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
		BindHost:        host,
		Port:            opts.port,
		ExposureMode:    config.NavivoxExposureLocal,
		AuthMode:        config.NavivoxAuthPairingToken,
		Token:           token,
		PublicConfirmed: false,
	}
	if err := config.ValidateNavivoxForRuntime(&runtimeCfg); err != nil {
		return err
	}
	for _, write := range []struct {
		key   string
		value string
	}{
		{"navivox.enabled", "true"},
		{"navivox.bind_host", runtimeCfg.BindHost},
		{"navivox.port", strconv.Itoa(runtimeCfg.Port)},
		{"navivox.exposure_mode", runtimeCfg.ExposureMode},
		{"navivox.auth_mode", runtimeCfg.AuthMode},
		{"navivox.public_confirmed", "false"},
	} {
		if err := config.WriteTOMLValue(config.ConfigPath(), write.key, write.value); err != nil {
			return err
		}
	}
	if err := config.WriteEnvValue(config.EnvPath(), "GORMES_NAVIVOX_TOKEN", token); err != nil {
		return err
	}

	baseURL, wsURL := navivoxConnectInfoURLs(runtimeCfg.BindHost, runtimeCfg.Port)
	descriptor := navivoxPairDescriptor(runtimeCfg, baseURL, wsURL)
	qrPath := filepath.Join(config.GormesHome(), "navivox", "pairing.png")
	if err := writeNavivoxPairQR(qrPath, descriptor); err != nil {
		return err
	}

	bridgeStop, bridgeDone, err := startNavivoxPairBridge(cmd.Context(), runtimeCfg)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Navivox pairing ready.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Connection")
	fmt.Fprintf(out, "  HTTP: %s\n", baseURL)
	fmt.Fprintf(out, "  WebSocket: %s\n", wsURL)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Pairing")
	if generatedToken {
		fmt.Fprintln(out, "  Token: generated and stored as GORMES_NAVIVOX_TOKEN in:")
	} else {
		fmt.Fprintln(out, "  Token: reused from GORMES_NAVIVOX_TOKEN in:")
	}
	fmt.Fprintf(out, "  %s\n", config.EnvPath())
	fmt.Fprintln(out, "  Pairing QR image:")
	fmt.Fprintf(out, "  %s\n", qrPath)
	fmt.Fprintln(out, "  Secret: the QR image embeds the local bridge URL and Navivox token.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Bridge")
	fmt.Fprintf(out, "  Local bridge URL: %s\n", baseURL)
	fmt.Fprintf(out, "  Local bridge listening: %s\n", baseURL)
	fmt.Fprintln(out, "  Lifecycle: keep this terminal open after Navivox connects.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Next steps")
	fmt.Fprintln(out, "  1. Open Navivox on Android and scan the QR image.")
	fmt.Fprintln(out, "  2. Finish provider, model, workspace, and channel setup in Navivox.")
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

func startNavivoxPairBridge(ctx context.Context, cfg config.NavivoxCfg) (context.CancelFunc, <-chan error, error) {
	ch, err := navivoxchannel.NewChannel(cfg, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("navivox pair: create local bridge: %w", err)
	}
	ln, err := net.Listen("tcp", net.JoinHostPort(strings.Trim(cfg.BindHost, "[]"), strconv.Itoa(cfg.Port)))
	if err != nil {
		return nil, nil, fmt.Errorf("navivox pair: start local bridge: %w", err)
	}
	bridgeCtx, stop := context.WithCancel(ctx)
	done := make(chan error, 1)
	inbox := make(chan gateway.InboundEvent, 16)
	server := &http.Server{Handler: ch.Handler(inbox)}
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
	return stop, done, nil
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
