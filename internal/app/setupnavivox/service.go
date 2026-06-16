package setupnavivox

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/skip2/go-qrcode"

	navivoxapp "github.com/TrebuchetDynamics/gormes-agent/internal/app/navivox"
	"github.com/TrebuchetDynamics/gormes-agent/internal/app/setupchoice"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/network/vpnhost"
)

const AppSourceURL = "https://github.com/TrebuchetDynamics/navivox-app"

var exposureChoices = []setupchoice.Choice{
	{ID: config.NavivoxExposureLocal, Label: "Local loopback only", Aliases: []string{"loopback"}},
	{ID: config.NavivoxExposureTailscale, Label: "Tailscale VPN"},
	{ID: config.NavivoxExposureWireGuard, Label: "WireGuard VPN", Aliases: []string{"wireguard", "wire_guard"}},
	{ID: config.NavivoxExposureVPN, Label: "Other VPN"},
	{ID: config.NavivoxExposurePublic, Label: "Public internet (requires typed confirmation)"},
}

var authChoices = []setupchoice.Choice{
	{ID: config.NavivoxAuthPairingToken, Label: "Pairing token"},
	{ID: config.NavivoxAuthStaticToken, Label: "Static token"},
	{ID: config.NavivoxAuthTailscaleIdentity, Label: "Tailscale identity"},
	{ID: config.NavivoxAuthTokenAndTailscaleIdentity, Label: "Token and Tailscale identity"},
}

// GatewayOptions are command-local Navivox setup seams. Cobra/root prompt
// details are injected by internal/platform/cli/gormescli.
type GatewayOptions struct {
	Context context.Context
	Out     io.Writer
	Config  config.Config

	AskYesNo     func(title, linePrompt string, defaultValue bool) (bool, bool, error)
	PromptChoice func(title, linePrompt, defaultID string, choices []setupchoice.Choice) (string, error)
	PromptString func(prompt, defaultValue string) (string, error)

	ParsePositiveInt func(string) (int, bool)
	VPNHostList      func(context.Context) ([]vpnhost.Host, error)

	GenerateSetupToken func() (string, error)
	NewGatewayID       func() (string, error)
	ValidateRuntime    func(*config.NavivoxCfg) error

	ConfigPath     func() string
	EnvPath        func() string
	GormesHome     func() string
	WriteTOMLValue func(path, key, value string) error

	WriteProfileChannelBinding func(ProfileChannelOptions) (ProfileChannelBinding, error)
	WriteGatewayTokenEnv       func(ProfileChannelBinding, string, string) error

	WritePairingQR func(path, descriptor string) error
	TerminalQR     func(descriptor string) (string, error)

	ProviderAuthConfigured func(config.Config) bool
}

type ProfileChannelOptions struct {
	ChannelID      string
	AllowedChats   []string
	AllowedUsers   []string
	RequireMention bool
	ToolProgress   string
}

type ProfileChannelBinding struct {
	ProfileID     string
	ChannelID     string
	CredentialID  string
	SecretEnvName string
	RegistryPath  string
}

func (opts GatewayOptions) withDefaults() GatewayOptions {
	if opts.Context == nil {
		opts.Context = context.Background()
	}
	if opts.Out == nil {
		opts.Out = io.Discard
	}
	if opts.ParsePositiveInt == nil {
		opts.ParsePositiveInt = parsePositiveInt
	}
	if opts.VPNHostList == nil {
		opts.VPNHostList = navivoxapp.DefaultVPNHostList
	}
	if opts.GenerateSetupToken == nil {
		opts.GenerateSetupToken = navivoxapp.GenerateSetupToken
	}
	if opts.NewGatewayID == nil {
		opts.NewGatewayID = config.NewNavivoxGatewayID
	}
	if opts.ValidateRuntime == nil {
		opts.ValidateRuntime = config.ValidateNavivoxForRuntime
	}
	if opts.ConfigPath == nil {
		opts.ConfigPath = config.ConfigPath
	}
	if opts.EnvPath == nil {
		opts.EnvPath = config.EnvPath
	}
	if opts.GormesHome == nil {
		opts.GormesHome = config.GormesHome
	}
	if opts.WriteTOMLValue == nil {
		opts.WriteTOMLValue = config.WriteTOMLValue
	}
	if opts.WritePairingQR == nil {
		opts.WritePairingQR = WritePairingQR
	}
	if opts.TerminalQR == nil {
		opts.TerminalQR = TerminalQR
	}
	if opts.ProviderAuthConfigured == nil {
		opts.ProviderAuthConfigured = func(config.Config) bool { return false }
	}
	return opts
}

// RunGateway runs the interactive Navivox Gateway Channel setup section.
func RunGateway(opts GatewayOptions) error {
	opts = opts.withDefaults()
	if opts.AskYesNo == nil {
		return fmt.Errorf("setup navivox: yes/no prompt is not configured")
	}
	if opts.PromptChoice == nil {
		return fmt.Errorf("setup navivox: choice prompt is not configured")
	}
	if opts.PromptString == nil {
		return fmt.Errorf("setup navivox: text prompt is not configured")
	}
	if opts.WriteProfileChannelBinding == nil {
		return fmt.Errorf("setup navivox: profile channel writer is not configured")
	}
	if opts.WriteGatewayTokenEnv == nil {
		return fmt.Errorf("setup navivox: gateway token writer is not configured")
	}

	cfg := opts.Config
	out := opts.Out
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Navivox Gateway Channel")
	fmt.Fprintln(out, "Native HTTP/WebSocket channel owned by `gormes gateway`; SSH remains break-glass only.")

	enabled, ok, err := opts.AskYesNo("Enable Navivox Gateway Channel?", "Enable Navivox Gateway Channel? [Y/n]: ", true)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("setup navivox: answer yes or no")
	}
	if !enabled {
		if err := opts.WriteTOMLValue(opts.ConfigPath(), "navivox.enabled", "false"); err != nil {
			return err
		}
		fmt.Fprintln(out, "Navivox gateway channel disabled.")
		fmt.Fprintln(out, "No firewall rules were changed.")
		return nil
	}

	exposureDefault := firstNonEmpty(cfg.Navivox.ExposureMode, config.NavivoxExposureLocal)
	exposureInput, err := opts.PromptChoice("Exposure mode", "Exposure mode (local/tailscale/wireguard/vpn/public) [local]: ", exposureDefault, exposureChoices)
	if err != nil {
		return err
	}
	exposureMode := setupchoice.NormalizeValue(exposureInput)
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
		confirm, err := opts.PromptString("Public exposure is discouraged. Type public to confirm: ", "")
		if err != nil {
			return err
		}
		if setupchoice.NormalizeValue(confirm) != config.NavivoxExposurePublic {
			return fmt.Errorf("setup navivox: public exposure was not confirmed")
		}
		publicConfirmed = true
	}

	currentBind := ""
	if cfg.Navivox.Enabled {
		currentBind = cfg.Navivox.BindHost
	}
	bindDefault := setupBindDefault(opts.Context, currentBind, exposureMode, opts.VPNHostList)
	bindHost, err := opts.PromptString(fmt.Sprintf("Bind host [%s]: ", bindDefault), bindDefault)
	if err != nil {
		return err
	}
	bindHost = strings.TrimSpace(bindHost)

	portDefault := cfg.Navivox.Port
	if portDefault == 0 {
		portDefault = config.NavivoxDefaultPort
	}
	portInput, err := opts.PromptString(fmt.Sprintf("Port [%d]: ", portDefault), strconv.Itoa(portDefault))
	if err != nil {
		return err
	}
	port, ok := opts.ParsePositiveInt(portInput)
	if !ok || port > 65535 {
		return fmt.Errorf("setup navivox: invalid port %q", portInput)
	}

	authDefault := firstNonEmpty(cfg.Navivox.AuthMode, config.NavivoxAuthPairingToken)
	authInput, err := opts.PromptChoice("Auth mode", "Auth mode (pairing_token/static_token/tailscale_identity/token_and_tailscale_identity) [pairing_token]: ", authDefault, authChoices)
	if err != nil {
		return err
	}
	authMode := setupchoice.NormalizeValue(authInput)
	if authMode == "" {
		authMode = config.NavivoxAuthPairingToken
	}
	switch authMode {
	case config.NavivoxAuthPairingToken, config.NavivoxAuthStaticToken, config.NavivoxAuthTailscaleIdentity, config.NavivoxAuthTokenAndTailscaleIdentity:
	default:
		return fmt.Errorf("setup navivox: unsupported auth mode %q", authInput)
	}

	var token string
	if TokenRequired(authMode) {
		token = strings.TrimSpace(cfg.Navivox.Token)
		if token == "" {
			generated, err := opts.GenerateSetupToken()
			if err != nil {
				return err
			}
			token = generated
		}
	}

	gatewayID := strings.TrimSpace(cfg.Navivox.GatewayID)
	if gatewayID == "" {
		generated, err := opts.NewGatewayID()
		if err != nil {
			return err
		}
		gatewayID = generated
	}
	gatewayLabel := strings.TrimSpace(cfg.Navivox.GatewayLabel)
	if gatewayLabel == "" {
		gatewayLabel = config.NavivoxDefaultGatewayLabel
	}

	var allowedIdentities string
	if authMode == config.NavivoxAuthTailscaleIdentity || authMode == config.NavivoxAuthTokenAndTailscaleIdentity {
		allowedInput, err := opts.PromptString("Allowed Tailscale identities (comma-separated, blank to allow Tailscale-authenticated clients): ", strings.Join(cfg.Navivox.AllowedTailnetIdentities, ","))
		if err != nil {
			return err
		}
		allowedIdentities = allowedInput
	}

	firewallRequested, ok, err := opts.AskYesNo("Record manual firewall-open intent?", "Record manual firewall-open intent? [n]: ", false)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("setup navivox: answer yes or no for firewall")
	}

	runtimeCfg := config.NavivoxCfg{
		Enabled:                  true,
		GatewayID:                gatewayID,
		GatewayLabel:             gatewayLabel,
		BindHost:                 bindHost,
		Port:                     port,
		ExposureMode:             exposureMode,
		AuthMode:                 authMode,
		Token:                    token,
		AllowedTailnetIdentities: CSV(allowedIdentities),
		PublicConfirmed:          publicConfirmed,
	}
	if err := opts.ValidateRuntime(&runtimeCfg); err != nil {
		return err
	}

	writes := []struct {
		key   string
		value string
	}{
		{"navivox.enabled", "true"},
		{"navivox.gateway_id", runtimeCfg.GatewayID},
		{"navivox.gateway_label", runtimeCfg.GatewayLabel},
		{"navivox.bind_host", runtimeCfg.BindHost},
		{"navivox.port", strconv.Itoa(runtimeCfg.Port)},
		{"navivox.exposure_mode", runtimeCfg.ExposureMode},
		{"navivox.auth_mode", runtimeCfg.AuthMode},
		{"navivox.public_confirmed", strconv.FormatBool(runtimeCfg.PublicConfirmed)},
	}
	for _, write := range writes {
		if err := opts.WriteTOMLValue(opts.ConfigPath(), write.key, write.value); err != nil {
			return err
		}
	}
	if err := opts.WriteTOMLValue(opts.ConfigPath(), "navivox.allowed_tailnet_identities", allowedIdentities); err != nil {
		return err
	}
	profileBinding, err := opts.WriteProfileChannelBinding(ProfileChannelOptions{ChannelID: "navivox"})
	if err != nil {
		return fmt.Errorf("setup navivox: write profile channel binding: %w", err)
	}
	if token != "" {
		if err := opts.WriteGatewayTokenEnv(profileBinding, "GORMES_NAVIVOX_TOKEN", token); err != nil {
			return err
		}
	}

	baseURL, wsURL := navivoxapp.NavivoxConnectInfoURLs(runtimeCfg.BindHost, runtimeCfg.Port)
	pairingURI, err := PairingURI(runtimeCfg)
	if err != nil {
		return err
	}
	qrPath := PairingQRPath(opts.GormesHome())
	if err := opts.WritePairingQR(qrPath, pairingURI); err != nil {
		return err
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Navivox gateway channel configured.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Connection")
	fmt.Fprintf(out, "  HTTP: %s\n", baseURL)
	fmt.Fprintf(out, "  WebSocket: %s\n", wsURL)
	fmt.Fprintln(out, "  Config:")
	fmt.Fprintf(out, "  %s\n", opts.ConfigPath())
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Pairing")
	if token != "" {
		fmt.Fprintln(out, "  Token: generated and stored as GORMES_NAVIVOX_TOKEN in:")
		fmt.Fprintf(out, "  %s\n", opts.EnvPath())
		fmt.Fprintf(out, "  Profile token: %s\n", profileBinding.SecretEnvName)
	}
	fmt.Fprintf(out, "  Profile channel: profiles.%s.channels.navivox\n", profileBinding.ProfileID)
	fmt.Fprintln(out, "  Pairing QR image:")
	fmt.Fprintf(out, "  %s\n", qrPath)
	fmt.Fprintln(out, "  Scan this QR from Navivox:")
	terminalQR, err := opts.TerminalQR(pairingURI)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(strings.TrimRight(terminalQR, "\n"), "\n") {
		fmt.Fprintf(out, "  %s\n", line)
	}
	fmt.Fprintln(out, "  QR payload includes the token when required; the raw token is not printed.")
	if token != "" {
		fmt.Fprintln(out, "  Secret: the QR image embeds the base URL and Navivox token.")
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Auth rules")
	if token != "" {
		fmt.Fprintln(out, "  REST: Authorization: Bearer <Navivox token>")
		fmt.Fprintln(out, "  WebSocket: Navivox token subprotocol, or Authorization header if supported.")
	} else {
		fmt.Fprintln(out, "  Token auth is disabled for this mode; Tailscale identity headers authorize requests.")
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Firewall")
	fmt.Fprintln(out, "  Status: no rules were changed by Gormes.")
	if firewallRequested {
		fmt.Fprintf(out, "  Operator request: recorded only; open %s:%d manually if needed.\n", runtimeCfg.BindHost, runtimeCfg.Port)
		fmt.Fprintln(out, "  Rollback: close that manual rule after testing.")
	} else {
		fmt.Fprintln(out, "  Operator request: none.")
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Get Navivox")
	fmt.Fprintf(out, "  Android app source: %s\n", AppSourceURL)
	fmt.Fprintf(out, "  Build/run from source: git clone %s && cd navivox-app && flutter run\n", AppSourceURL)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Next steps")
	fmt.Fprintln(out, "  1. Install or open Navivox on Android.")
	fmt.Fprintln(out, "  2. Scan the QR above, or open the QR image from:")
	fmt.Fprintf(out, "  %s\n", qrPath)
	if command := ProviderSetupCommand(cfg, opts.ProviderAuthConfigured(cfg)); command != "" {
		fmt.Fprintf(out, "  3. Configure provider before starting gateway: %s\n", command)
		fmt.Fprintln(out, "  4. Then start gateway: gormes gateway")
	} else {
		fmt.Fprintln(out, "  3. Start gateway: gormes gateway")
	}
	return nil
}

// PairingURI builds the Navivox setup pairing descriptor for the configured endpoint.
func PairingURI(cfg config.NavivoxCfg) (string, error) {
	baseURL, webSocketURL := urls(cfg.BindHost, cfg.Port)
	values := url.Values{}
	values.Set("base_url", baseURL)
	values.Set("websocket_url", webSocketURL)
	values.Set("capabilities_url", baseURL+"/v1/navivox/capabilities")
	values.Set("auth_mode", strings.TrimSpace(cfg.AuthMode))
	values.Set("exposure_mode", strings.TrimSpace(cfg.ExposureMode))
	tokenRequired := TokenRequired(cfg.AuthMode)
	values.Set("token_required", fmt.Sprintf("%t", tokenRequired))
	if tokenRequired {
		if strings.TrimSpace(cfg.Token) == "" {
			return "", fmt.Errorf("setup navivox: token auth selected but token is empty")
		}
		values.Set("rest_token", cfg.Token)
	}
	return (&url.URL{Scheme: "navivox", Host: "connect", RawQuery: values.Encode()}).String(), nil
}

// TokenRequired reports whether the auth mode needs a REST/WebSocket token.
func TokenRequired(authMode string) bool {
	switch strings.TrimSpace(authMode) {
	case config.NavivoxAuthPairingToken, config.NavivoxAuthStaticToken, config.NavivoxAuthTokenAndTailscaleIdentity:
		return true
	default:
		return false
	}
}

// BindDefault chooses a setup bind host from current config, exposure mode, and VPN hosts.
func BindDefault(current, exposureMode string, hosts []vpnhost.Host) string {
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
		return vpnBindDefault(hosts, func(h vpnhost.Host) bool { return h.Kind == vpnhost.KindTailscale })
	case config.NavivoxExposureWireGuard:
		return vpnBindDefault(hosts, func(h vpnhost.Host) bool { return h.Kind == vpnhost.KindWireGuard })
	case config.NavivoxExposureVPN:
		return vpnBindDefault(hosts, func(vpnhost.Host) bool { return true })
	default:
		return config.NavivoxDefaultBindHost
	}
}

func setupBindDefault(ctx context.Context, current, exposureMode string, lister func(context.Context) ([]vpnhost.Host, error)) string {
	hosts, _ := lister(ctx)
	return BindDefault(current, exposureMode, hosts)
}

func urls(host string, port int) (baseURL, webSocketURL string) {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	hostPort := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	baseURL = "http://" + hostPort
	webSocketURL = "ws://" + hostPort + "/v1/navivox/stream"
	return baseURL, webSocketURL
}

func vpnBindDefault(hosts []vpnhost.Host, match func(vpnhost.Host) bool) string {
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

func PairingQRPath(home string) string {
	return filepath.Join(home, "cache", "navivox", "pairing.png")
}

func WritePairingQR(path, descriptor string) error {
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

func TerminalQR(descriptor string) (string, error) {
	if strings.TrimSpace(descriptor) == "" {
		return "", fmt.Errorf("setup navivox: pairing descriptor is empty")
	}
	qr, err := qrcode.New(descriptor, qrcode.Medium)
	if err != nil {
		return "", fmt.Errorf("setup navivox: encode terminal QR: %w", err)
	}
	return qr.ToSmallString(false), nil
}

func ProviderSetupCommand(cfg config.Config, authConfigured bool) string {
	if strings.TrimSpace(cfg.Hermes.Endpoint) == "" || !authConfigured {
		return "gormes setup provider"
	}
	if strings.TrimSpace(cfg.Hermes.Model) == "" {
		return "gormes setup model"
	}
	return ""
}

func CSV(value string) []string {
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

func parsePositiveInt(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
