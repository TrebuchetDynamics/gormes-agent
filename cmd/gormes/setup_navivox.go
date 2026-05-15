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
	"github.com/TrebuchetDynamics/gormes-agent/internal/network/vpnhost"
)

// vpnhostList is the seam test code can override to inject deterministic
// VPN host enumeration; production callers go through the real CLIs.
var vpnhostList = vpnhost.List

func runSetupNavivoxGateway(cmd *cobra.Command, cfg config.Config) error {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Navivox Gateway Channel")
	fmt.Fprintln(out, "Native HTTP/WebSocket channel owned by `gormes gateway`; SSH remains break-glass only.")

	enabledInput, err := promptString(cmd, "Enable Navivox Gateway Channel? [Y/n]: ", "yes")
	if err != nil {
		return err
	}
	enabled, ok := parseSetupYesNo(enabledInput, true)
	if !ok {
		return fmt.Errorf("setup navivox: answer yes or no")
	}
	if !enabled {
		if err := config.WriteTOMLValue(config.ConfigPath(), "navivox.enabled", "false"); err != nil {
			return err
		}
		fmt.Fprintln(out, "Navivox gateway channel disabled.")
		fmt.Fprintln(out, "No firewall rules were changed.")
		return nil
	}

	exposureDefault := firstNonEmptySetup(cfg.Navivox.ExposureMode, config.NavivoxExposureLocal)
	exposureInput, err := promptString(cmd, "Exposure mode (local/tailscale/public) [local]: ", exposureDefault)
	if err != nil {
		return err
	}
	exposureMode := normalizeSetupChoice(exposureInput)
	if exposureMode == "" {
		exposureMode = config.NavivoxExposureLocal
	}
	switch exposureMode {
	case config.NavivoxExposureLocal, config.NavivoxExposureTailscale, config.NavivoxExposurePublic:
	default:
		return fmt.Errorf("setup navivox: unsupported exposure mode %q", exposureInput)
	}

	publicConfirmed := false
	if exposureMode == config.NavivoxExposurePublic {
		confirm, err := promptString(cmd, "Public exposure is discouraged. Type public to confirm: ", "")
		if err != nil {
			return err
		}
		if normalizeSetupChoice(confirm) != config.NavivoxExposurePublic {
			return fmt.Errorf("setup navivox: public exposure was not confirmed")
		}
		publicConfirmed = true
	}

	currentBind := ""
	if cfg.Navivox.Enabled {
		currentBind = cfg.Navivox.BindHost
	}
	bindDefault := navivoxSetupBindDefault(cmd.Context(), currentBind, exposureMode)
	bindHost, err := promptString(cmd, fmt.Sprintf("Bind host [%s]: ", bindDefault), bindDefault)
	if err != nil {
		return err
	}
	bindHost = strings.TrimSpace(bindHost)

	portDefault := cfg.Navivox.Port
	if portDefault == 0 {
		portDefault = config.NavivoxDefaultPort
	}
	portInput, err := promptString(cmd, fmt.Sprintf("Port [%d]: ", portDefault), strconv.Itoa(portDefault))
	if err != nil {
		return err
	}
	port, ok := parsePositiveInt(portInput)
	if !ok || port > 65535 {
		return fmt.Errorf("setup navivox: invalid port %q", portInput)
	}

	authDefault := firstNonEmptySetup(cfg.Navivox.AuthMode, config.NavivoxAuthPairingToken)
	authInput, err := promptString(cmd, "Auth mode (pairing_token/static_token/tailscale_identity) [pairing_token]: ", authDefault)
	if err != nil {
		return err
	}
	authMode := normalizeSetupChoice(authInput)
	if authMode == "" {
		authMode = config.NavivoxAuthPairingToken
	}
	switch authMode {
	case config.NavivoxAuthPairingToken, config.NavivoxAuthStaticToken, config.NavivoxAuthTailscaleIdentity:
	default:
		return fmt.Errorf("setup navivox: unsupported auth mode %q", authInput)
	}

	var token string
	if authMode == config.NavivoxAuthPairingToken || authMode == config.NavivoxAuthStaticToken {
		token = strings.TrimSpace(cfg.Navivox.Token)
		if token == "" {
			generated, err := generateNavivoxSetupToken()
			if err != nil {
				return err
			}
			token = generated
		}
	}

	var allowedIdentities string
	if authMode == config.NavivoxAuthTailscaleIdentity {
		allowedInput, err := promptString(cmd, "Allowed Tailscale identities (comma-separated, blank to allow Tailscale-authenticated clients): ", strings.Join(cfg.Navivox.AllowedTailnetIdentities, ","))
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
		return fmt.Errorf("setup navivox: answer yes or no for firewall")
	}

	runtimeCfg := config.NavivoxCfg{
		Enabled:                  true,
		BindHost:                 bindHost,
		Port:                     port,
		ExposureMode:             exposureMode,
		AuthMode:                 authMode,
		Token:                    token,
		AllowedTailnetIdentities: parseSetupCSV(allowedIdentities),
		PublicConfirmed:          publicConfirmed,
	}
	if err := config.ValidateNavivoxForRuntime(&runtimeCfg); err != nil {
		return err
	}

	writes := []struct {
		key   string
		value string
	}{
		{"navivox.enabled", "true"},
		{"navivox.bind_host", runtimeCfg.BindHost},
		{"navivox.port", strconv.Itoa(runtimeCfg.Port)},
		{"navivox.exposure_mode", runtimeCfg.ExposureMode},
		{"navivox.auth_mode", runtimeCfg.AuthMode},
		{"navivox.public_confirmed", strconv.FormatBool(runtimeCfg.PublicConfirmed)},
	}
	for _, write := range writes {
		if err := config.WriteTOMLValue(config.ConfigPath(), write.key, write.value); err != nil {
			return err
		}
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), "navivox.allowed_tailnet_identities", allowedIdentities); err != nil {
		return err
	}
	if token != "" {
		if err := config.WriteEnvValue(config.EnvPath(), "GORMES_NAVIVOX_TOKEN", token); err != nil {
			return err
		}
	}

	baseURL := fmt.Sprintf("http://%s:%d", runtimeCfg.BindHost, runtimeCfg.Port)
	wsURL := fmt.Sprintf("ws://%s:%d/v1/navivox/stream", runtimeCfg.BindHost, runtimeCfg.Port)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Navivox gateway channel configured.")
	fmt.Fprintf(out, "HTTP base URL: %s\n", baseURL)
	fmt.Fprintf(out, "WebSocket URL: %s\n", wsURL)
	fmt.Fprintf(out, "Config: %s\n", config.ConfigPath())
	if token != "" {
		fmt.Fprintf(out, "Pairing token: generated and stored in %s as GORMES_NAVIVOX_TOKEN.\n", config.EnvPath())
	}
	fmt.Fprintln(out, "Firewall: no rules were changed.")
	if firewallRequested {
		fmt.Fprintf(out, "Firewall request recorded as operator intent only; open %s:%d manually and keep rollback documented.\n", runtimeCfg.BindHost, runtimeCfg.Port)
	}
	fmt.Fprintln(out, "Start gateway: gormes gateway")
	return nil
}

func navivoxSetupBindDefault(ctx context.Context, current, exposureMode string) string {
	current = strings.TrimSpace(current)
	if current != "" {
		return current
	}
	switch exposureMode {
	case config.NavivoxExposureLocal:
		return config.NavivoxDefaultBindHost
	case config.NavivoxExposurePublic:
		return "0.0.0.0"
	case config.NavivoxExposureTailscale:
		hosts, _ := vpnhostList(ctx)
		for _, h := range hosts {
			if h.Kind == vpnhost.KindTailscale && h.IPv4 != "" {
				return h.IPv4
			}
		}
		return config.NavivoxDefaultBindHost
	default:
		return config.NavivoxDefaultBindHost
	}
}

func generateNavivoxSetupToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("setup navivox: generate token: %w", err)
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
