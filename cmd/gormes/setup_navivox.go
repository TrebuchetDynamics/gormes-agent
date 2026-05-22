package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/skip2/go-qrcode"
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
	exposureInput, err := promptString(cmd, "Exposure mode (local/tailscale/wireguard/vpn/public) [local]: ", exposureDefault)
	if err != nil {
		return err
	}
	exposureMode := normalizeSetupChoice(exposureInput)
	if exposureMode == "" {
		exposureMode = config.NavivoxExposureLocal
	}
	switch exposureMode {
	case config.NavivoxExposureLocal,
		config.NavivoxExposureTailscale,
		config.NavivoxExposureWireGuard,
		config.NavivoxExposureVPN,
		config.NavivoxExposurePublic:
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
	authInput, err := promptString(cmd, "Auth mode (pairing_token/static_token/tailscale_identity/token_and_tailscale_identity) [pairing_token]: ", authDefault)
	if err != nil {
		return err
	}
	authMode := normalizeSetupChoice(authInput)
	if authMode == "" {
		authMode = config.NavivoxAuthPairingToken
	}
	switch authMode {
	case config.NavivoxAuthPairingToken, config.NavivoxAuthStaticToken, config.NavivoxAuthTailscaleIdentity, config.NavivoxAuthTokenAndTailscaleIdentity:
	default:
		return fmt.Errorf("setup navivox: unsupported auth mode %q", authInput)
	}

	var token string
	if authMode == config.NavivoxAuthPairingToken || authMode == config.NavivoxAuthStaticToken || authMode == config.NavivoxAuthTokenAndTailscaleIdentity {
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
	if authMode == config.NavivoxAuthTailscaleIdentity || authMode == config.NavivoxAuthTokenAndTailscaleIdentity {
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

	baseURL, wsURL := navivoxConnectInfoURLs(runtimeCfg.BindHost, runtimeCfg.Port)
	pairingURI, err := navivoxSetupPairingURI(runtimeCfg)
	if err != nil {
		return err
	}
	qrPath := navivoxSetupPairingQRPath()
	if err := writeNavivoxSetupPairingQR(qrPath, pairingURI); err != nil {
		return err
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Navivox gateway channel configured.")
	fmt.Fprintf(out, "HTTP base URL: %s\n", baseURL)
	fmt.Fprintf(out, "WebSocket URL: %s\n", wsURL)
	fmt.Fprintf(out, "Config: %s\n", config.ConfigPath())
	if token != "" {
		fmt.Fprintf(out, "Pairing token: generated and stored in %s as GORMES_NAVIVOX_TOKEN.\n", config.EnvPath())
	}
	fmt.Fprintf(out, "Pairing QR image: %s\n", qrPath)
	fmt.Fprintln(out, "REST/WebSocket auth rules:")
	if token != "" {
		fmt.Fprintln(out, "  - REST: send Authorization: Bearer <Navivox token> on /v1/navivox/* requests.")
		fmt.Fprintln(out, "  - WebSocket: use the Navivox token subprotocol, or an Authorization header when the client supports headers.")
		fmt.Fprintln(out, "  - Treat the QR image as secret; it embeds the base URL and Navivox token.")
	} else {
		fmt.Fprintln(out, "  - Token auth is disabled for this mode; Tailscale identity headers authorize requests.")
	}
	fmt.Fprintln(out, "Firewall: no rules were changed.")
	if firewallRequested {
		fmt.Fprintf(out, "Firewall request recorded as operator intent only; open %s:%d manually and keep rollback documented.\n", runtimeCfg.BindHost, runtimeCfg.Port)
	}
	fmt.Fprintln(out, "Start gateway: gormes gateway")
	return nil
}

func navivoxSetupPairingQRPath() string {
	return filepath.Join(config.GormesHome(), "navivox", "pairing.png")
}

func writeNavivoxSetupPairingQR(path, descriptor string) error {
	if strings.TrimSpace(descriptor) == "" {
		return fmt.Errorf("setup navivox: pairing descriptor is empty")
	}
	pngBytes, err := qrcode.Encode(descriptor, qrcode.Medium, 512)
	if err != nil {
		return fmt.Errorf("setup navivox: encode pairing QR: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("setup navivox: create QR directory: %w", err)
	}
	if err := os.WriteFile(path, pngBytes, 0o600); err != nil {
		return fmt.Errorf("setup navivox: write pairing QR: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("setup navivox: secure pairing QR: %w", err)
	}
	return nil
}

func navivoxSetupPairingURI(cfg config.NavivoxCfg) (string, error) {
	baseURL, webSocketURL := navivoxConnectInfoURLs(cfg.BindHost, cfg.Port)
	values := url.Values{}
	values.Set("base_url", baseURL)
	values.Set("websocket_url", webSocketURL)
	values.Set("auth_mode", strings.TrimSpace(cfg.AuthMode))
	values.Set("exposure_mode", strings.TrimSpace(cfg.ExposureMode))
	tokenRequired := cfg.AuthMode == config.NavivoxAuthPairingToken ||
		cfg.AuthMode == config.NavivoxAuthStaticToken ||
		cfg.AuthMode == config.NavivoxAuthTokenAndTailscaleIdentity
	values.Set("token_required", strconv.FormatBool(tokenRequired))
	if tokenRequired {
		if strings.TrimSpace(cfg.Token) == "" {
			return "", fmt.Errorf("setup navivox: token auth selected but token is empty")
		}
		values.Set("rest_token", cfg.Token)
	}
	return (&url.URL{Scheme: "navivox", Host: "connect", RawQuery: values.Encode()}).String(), nil
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
		return navivoxSetupVPNBindDefault(ctx, func(h vpnhost.Host) bool {
			return h.Kind == vpnhost.KindTailscale
		})
	case config.NavivoxExposureWireGuard:
		return navivoxSetupVPNBindDefault(ctx, func(h vpnhost.Host) bool {
			return h.Kind == vpnhost.KindWireGuard
		})
	case config.NavivoxExposureVPN:
		return navivoxSetupVPNBindDefault(ctx, func(vpnhost.Host) bool { return true })
	default:
		return config.NavivoxDefaultBindHost
	}
}

func navivoxSetupVPNBindDefault(ctx context.Context, match func(vpnhost.Host) bool) string {
	hosts, _ := vpnhostList(ctx)
	for _, h := range hosts {
		if !match(h) {
			continue
		}
		if h.IPv4 != "" {
			return h.IPv4
		}
	}
	return config.NavivoxDefaultBindHost
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
