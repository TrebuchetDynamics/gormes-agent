package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func runSetupNaviboxGateway(cmd *cobra.Command, cfg config.Config) error {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Navibox Gateway Channel")
	fmt.Fprintln(out, "Native HTTP/WebSocket channel owned by `gormes gateway`; SSH remains break-glass only.")

	enabledInput, err := promptString(cmd, "Enable Navibox Gateway Channel? [Y/n]: ", "yes")
	if err != nil {
		return err
	}
	enabled, ok := parseSetupYesNo(enabledInput, true)
	if !ok {
		return fmt.Errorf("setup navibox: answer yes or no")
	}
	if !enabled {
		if err := config.WriteTOMLValue(config.ConfigPath(), "navibox.enabled", "false"); err != nil {
			return err
		}
		fmt.Fprintln(out, "Navibox gateway channel disabled.")
		fmt.Fprintln(out, "No firewall rules were changed.")
		return nil
	}

	exposureDefault := firstNonEmptySetup(cfg.Navibox.ExposureMode, config.NaviboxExposureLocal)
	exposureInput, err := promptString(cmd, "Exposure mode (local/tailscale/public) [local]: ", exposureDefault)
	if err != nil {
		return err
	}
	exposureMode := normalizeSetupChoice(exposureInput)
	if exposureMode == "" {
		exposureMode = config.NaviboxExposureLocal
	}
	switch exposureMode {
	case config.NaviboxExposureLocal, config.NaviboxExposureTailscale, config.NaviboxExposurePublic:
	default:
		return fmt.Errorf("setup navibox: unsupported exposure mode %q", exposureInput)
	}

	publicConfirmed := false
	if exposureMode == config.NaviboxExposurePublic {
		confirm, err := promptString(cmd, "Public exposure is discouraged. Type public to confirm: ", "")
		if err != nil {
			return err
		}
		if normalizeSetupChoice(confirm) != config.NaviboxExposurePublic {
			return fmt.Errorf("setup navibox: public exposure was not confirmed")
		}
		publicConfirmed = true
	}

	currentBind := ""
	if cfg.Navibox.Enabled {
		currentBind = cfg.Navibox.BindHost
	}
	bindDefault := naviboxSetupBindDefault(cmd.Context(), currentBind, exposureMode)
	bindHost, err := promptString(cmd, fmt.Sprintf("Bind host [%s]: ", bindDefault), bindDefault)
	if err != nil {
		return err
	}
	bindHost = strings.TrimSpace(bindHost)

	portDefault := cfg.Navibox.Port
	if portDefault == 0 {
		portDefault = config.NaviboxDefaultPort
	}
	portInput, err := promptString(cmd, fmt.Sprintf("Port [%d]: ", portDefault), strconv.Itoa(portDefault))
	if err != nil {
		return err
	}
	port, ok := parsePositiveInt(portInput)
	if !ok || port > 65535 {
		return fmt.Errorf("setup navibox: invalid port %q", portInput)
	}

	authDefault := firstNonEmptySetup(cfg.Navibox.AuthMode, config.NaviboxAuthPairingToken)
	authInput, err := promptString(cmd, "Auth mode (pairing_token/static_token/tailscale_identity) [pairing_token]: ", authDefault)
	if err != nil {
		return err
	}
	authMode := normalizeSetupChoice(authInput)
	if authMode == "" {
		authMode = config.NaviboxAuthPairingToken
	}
	switch authMode {
	case config.NaviboxAuthPairingToken, config.NaviboxAuthStaticToken, config.NaviboxAuthTailscaleIdentity:
	default:
		return fmt.Errorf("setup navibox: unsupported auth mode %q", authInput)
	}

	var token string
	if authMode == config.NaviboxAuthPairingToken || authMode == config.NaviboxAuthStaticToken {
		token = strings.TrimSpace(cfg.Navibox.Token)
		if token == "" {
			generated, err := generateNaviboxSetupToken()
			if err != nil {
				return err
			}
			token = generated
		}
	}

	var allowedIdentities string
	if authMode == config.NaviboxAuthTailscaleIdentity {
		allowedInput, err := promptString(cmd, "Allowed Tailscale identities (comma-separated, blank to allow Tailscale-authenticated clients): ", strings.Join(cfg.Navibox.AllowedTailnetIdentities, ","))
		if err != nil {
			return err
		}
		allowedIdentities = allowedInput
	}

	firewallInput, err := promptString(cmd, "Open firewall rule for this port now? [n]: ", "no")
	if err != nil {
		return err
	}
	firewallRequested, ok := parseSetupYesNo(firewallInput, false)
	if !ok {
		return fmt.Errorf("setup navibox: answer yes or no for firewall")
	}

	runtimeCfg := config.NaviboxCfg{
		Enabled:                  true,
		BindHost:                 bindHost,
		Port:                     port,
		ExposureMode:             exposureMode,
		AuthMode:                 authMode,
		Token:                    token,
		AllowedTailnetIdentities: parseSetupCSV(allowedIdentities),
		PublicConfirmed:          publicConfirmed,
	}
	if err := config.ValidateNaviboxForRuntime(&runtimeCfg); err != nil {
		return err
	}

	writes := []struct {
		key   string
		value string
	}{
		{"navibox.enabled", "true"},
		{"navibox.bind_host", runtimeCfg.BindHost},
		{"navibox.port", strconv.Itoa(runtimeCfg.Port)},
		{"navibox.exposure_mode", runtimeCfg.ExposureMode},
		{"navibox.auth_mode", runtimeCfg.AuthMode},
		{"navibox.public_confirmed", strconv.FormatBool(runtimeCfg.PublicConfirmed)},
	}
	for _, write := range writes {
		if err := config.WriteTOMLValue(config.ConfigPath(), write.key, write.value); err != nil {
			return err
		}
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "navibox.allowed_tailnet_identities", allowedIdentities); err != nil {
		return err
	}
	if token != "" {
		if err := config.WriteEnvValue(config.EnvPath(), "GORMES_NAVIBOX_TOKEN", token); err != nil {
			return err
		}
	}

	baseURL := fmt.Sprintf("http://%s:%d", runtimeCfg.BindHost, runtimeCfg.Port)
	wsURL := fmt.Sprintf("ws://%s:%d/v1/navibox/stream", runtimeCfg.BindHost, runtimeCfg.Port)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Navibox gateway channel configured.")
	fmt.Fprintf(out, "HTTP base URL: %s\n", baseURL)
	fmt.Fprintf(out, "WebSocket URL: %s\n", wsURL)
	fmt.Fprintf(out, "Config: %s\n", config.ConfigPath())
	if token != "" {
		fmt.Fprintf(out, "Pairing token: generated and stored in %s as GORMES_NAVIBOX_TOKEN.\n", config.EnvPath())
	}
	fmt.Fprintln(out, "Firewall: no rules were changed.")
	if firewallRequested {
		fmt.Fprintf(out, "Firewall request recorded as operator intent only; open %s:%d manually and keep rollback documented.\n", runtimeCfg.BindHost, runtimeCfg.Port)
	}
	fmt.Fprintln(out, "Start gateway: gormes gateway")
	return nil
}

func naviboxSetupBindDefault(ctx context.Context, current, exposureMode string) string {
	current = strings.TrimSpace(current)
	if current != "" {
		return current
	}
	switch exposureMode {
	case config.NaviboxExposureLocal:
		return config.NaviboxDefaultBindHost
	case config.NaviboxExposurePublic:
		return "0.0.0.0"
	case config.NaviboxExposureTailscale:
		host, source, err := resolveNavivoxPairHost(ctx, "")
		if err == nil && source == "tailscale" && strings.TrimSpace(host) != "" {
			return host
		}
		return config.NaviboxDefaultBindHost
	default:
		return config.NaviboxDefaultBindHost
	}
}

func generateNaviboxSetupToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("setup navibox: generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func parseSetupYesNo(value string, defaultValue bool) (bool, bool) {
	value = normalizeSetupChoice(value)
	if value == "" {
		return defaultValue, true
	}
	switch value {
	case "y", "yes", "true", "1", "on":
		return true, true
	case "n", "no", "false", "0", "off":
		return false, true
	default:
		return false, false
	}
}

func parseSetupCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
