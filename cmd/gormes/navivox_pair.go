package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/skip2/go-qrcode"
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
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
		Use:          "pair",
		Short:        "Create a local Navivox pairing handoff",
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

	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Navivox pairing ready.")
	fmt.Fprintf(out, "Local bridge URL: %s\n", baseURL)
	fmt.Fprintf(out, "WebSocket URL: %s\n", wsURL)
	if generatedToken {
		fmt.Fprintf(out, "Pairing token: generated and stored in %s as GORMES_NAVIVOX_TOKEN.\n", config.EnvPath())
	} else {
		fmt.Fprintf(out, "Pairing token: reused from %s as GORMES_NAVIVOX_TOKEN.\n", config.EnvPath())
	}
	fmt.Fprintf(out, "Pairing QR image: %s\n", qrPath)
	fmt.Fprintln(out, "Open Navivox on Android and scan the QR.")
	fmt.Fprintln(out, "Start local bridge: gormes gateway")
	if opts.noWait {
		fmt.Fprintln(out, "Waiting for Navivox connection skipped (--no-wait).")
		return nil
	}
	fmt.Fprintln(out, "Waiting for Navivox connection... Press Ctrl-C to stop.")
	<-cmd.Context().Done()
	return nil
}

func navivoxPairDescriptor(cfg config.NavivoxCfg, baseURL, wsURL string) string {
	values := url.Values{}
	values.Set("base_url", baseURL)
	values.Set("websocket_url", wsURL)
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
