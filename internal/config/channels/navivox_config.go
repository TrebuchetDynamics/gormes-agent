package channels

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"regexp"
	"strings"
)

const (
	NavivoxDefaultBindHost     = "127.0.0.1"
	NavivoxDefaultPort         = 8765
	NavivoxDefaultGatewayLabel = "Gormes gateway"

	NavivoxExposureLocal     = "local"
	NavivoxExposureTailscale = "tailscale"
	NavivoxExposureWireGuard = "wireguard"
	NavivoxExposureVPN       = "vpn"
	NavivoxExposurePublic    = "public"

	NavivoxAuthPairingToken              = "pairing_token"
	NavivoxAuthStaticToken               = "static_token"
	NavivoxAuthTailscaleIdentity         = "tailscale_identity"
	NavivoxAuthTokenAndTailscaleIdentity = "token_and_tailscale_identity"

	NavivoxMinExposedTokenLength        = 32
	NavivoxMinExposedTokenDistinctChars = 16
)

var (
	navivoxIDPattern        = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
	navivoxGatewayIDPattern = regexp.MustCompile(`^gw_[a-f0-9]{32}$`)
)

// NewNavivoxGatewayID creates the opaque public Gormes Gateway identity used
// by Navivox pairing/reconnect metadata. It is intentionally random and never
// derived from tokens, URLs, machine names, usernames, or paths.
func NewNavivoxGatewayID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("config: generate navivox.gateway_id: %w", err)
	}
	return "gw_" + hex.EncodeToString(raw[:]), nil
}

// NavivoxCfg configures the native gateway-owned HTTP/WebSocket channel used
// by the Flutter Navivox app. The disabled zero value is intentionally safe.
type NavivoxCfg struct {
	Enabled                  bool                        `toml:"enabled" yaml:"enabled"`
	GatewayID                string                      `toml:"gateway_id" yaml:"gateway_id"`
	GatewayLabel             string                      `toml:"gateway_label" yaml:"gateway_label"`
	BindHost                 string                      `toml:"bind_host" yaml:"bind_host"`
	Port                     int                         `toml:"port" yaml:"port"`
	ExposureMode             string                      `toml:"exposure_mode" yaml:"exposure_mode"`
	AuthMode                 string                      `toml:"auth_mode" yaml:"auth_mode"`
	Token                    string                      `toml:"token" yaml:"token"`
	AllowOrigins             []string                    `toml:"allow_origins" yaml:"allow_origins"`
	AllowedTailnetIdentities []string                    `toml:"allowed_tailnet_identities" yaml:"allowed_tailnet_identities"`
	PublicConfirmed          bool                        `toml:"public_confirmed" yaml:"public_confirmed"`
	Servers                  map[string]NavivoxServerCfg `toml:"servers" yaml:"servers"`
}

type NavivoxServerCfg struct {
	Enabled      bool     `toml:"enabled" yaml:"enabled"`
	Bind         string   `toml:"bind" yaml:"bind"`
	Profiles     []string `toml:"profiles" yaml:"profiles"`
	Transports   []string `toml:"transports" yaml:"transports"`
	Capabilities []string `toml:"capabilities" yaml:"capabilities"`
}

func NormalizeNavivoxConfig(cfg *NavivoxCfg) error {
	cfg.GatewayID = strings.TrimSpace(cfg.GatewayID)
	cfg.GatewayLabel = strings.TrimSpace(cfg.GatewayLabel)
	if cfg.GatewayLabel == "" {
		cfg.GatewayLabel = NavivoxDefaultGatewayLabel
	}
	cfg.BindHost = strings.TrimSpace(cfg.BindHost)
	if cfg.BindHost == "" {
		cfg.BindHost = NavivoxDefaultBindHost
	}
	if cfg.Port == 0 {
		cfg.Port = NavivoxDefaultPort
	}
	cfg.ExposureMode = strings.ToLower(strings.TrimSpace(cfg.ExposureMode))
	if cfg.ExposureMode == "" {
		cfg.ExposureMode = NavivoxExposureLocal
	}
	cfg.AuthMode = strings.ToLower(strings.TrimSpace(cfg.AuthMode))
	if cfg.AuthMode == "" {
		cfg.AuthMode = NavivoxAuthPairingToken
	}
	cfg.Token = strings.TrimSpace(cfg.Token)
	cfg.AllowOrigins = compactStrings(cfg.AllowOrigins)
	cfg.AllowedTailnetIdentities = compactStrings(cfg.AllowedTailnetIdentities)
	if err := normalizeNavivoxServers(cfg); err != nil {
		return err
	}

	if cfg.GatewayID != "" && !navivoxGatewayIDPattern.MatchString(cfg.GatewayID) {
		return fmt.Errorf("config: navivox.gateway_id must be a generated gw_ identity")
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("config: navivox.port must be between 1 and 65535, got %d", cfg.Port)
	}
	switch cfg.ExposureMode {
	case NavivoxExposureLocal,
		NavivoxExposureTailscale,
		NavivoxExposureWireGuard,
		NavivoxExposureVPN,
		NavivoxExposurePublic:
	default:
		return fmt.Errorf("config: navivox.exposure_mode %q is invalid; want local, tailscale, wireguard, vpn, or public", cfg.ExposureMode)
	}
	switch cfg.AuthMode {
	case NavivoxAuthPairingToken, NavivoxAuthStaticToken, NavivoxAuthTokenAndTailscaleIdentity:
		if cfg.Enabled && cfg.Token == "" {
			return fmt.Errorf("config: navivox.token is required when navivox.enabled=true and auth_mode=%s", cfg.AuthMode)
		}
	case NavivoxAuthTailscaleIdentity:
	default:
		return fmt.Errorf("config: navivox.auth_mode %q is invalid; want pairing_token, static_token, tailscale_identity, or token_and_tailscale_identity", cfg.AuthMode)
	}
	if !cfg.Enabled {
		return nil
	}
	if cfg.AuthMode == NavivoxAuthTailscaleIdentity {
		return fmt.Errorf("config: navivox.auth_mode=tailscale_identity is not allowed as standalone Navivox auth; use token auth or token_and_tailscale_identity")
	}
	if err := navivoxValidateExposedToken(cfg); err != nil {
		return err
	}
	if cfg.ExposureMode != NavivoxExposureLocal && navivoxWildcardOrigin(cfg.AllowOrigins) {
		return fmt.Errorf("config: navivox.allow_origins must not include * when exposure_mode=%s; list trusted Navivox browser origins explicitly", cfg.ExposureMode)
	}
	if navivoxWildcardHost(cfg.BindHost) && cfg.ExposureMode != NavivoxExposurePublic {
		return fmt.Errorf("config: navivox.bind_host %q requires navivox.exposure_mode=public and explicit confirmation", cfg.BindHost)
	}
	if cfg.ExposureMode == NavivoxExposureLocal && !navivoxLoopbackHost(cfg.BindHost) {
		return fmt.Errorf("config: navivox.exposure_mode=local requires loopback bind_host, got %q", cfg.BindHost)
	}
	if cfg.ExposureMode == NavivoxExposurePublic && !cfg.PublicConfirmed {
		return fmt.Errorf("config: navivox.exposure_mode=public requires navivox.public_confirmed=true")
	}
	return nil
}

func ValidateNavivoxForRuntime(cfg *NavivoxCfg) error {
	return NormalizeNavivoxConfig(cfg)
}

func normalizeNavivoxServers(cfg *NavivoxCfg) error {
	if len(cfg.Servers) == 0 {
		cfg.Servers = nil
		return nil
	}
	servers := make(map[string]NavivoxServerCfg, len(cfg.Servers))
	for id, server := range cfg.Servers {
		normalizedID := strings.ToLower(strings.TrimSpace(id))
		if normalizedID != id || !navivoxIDPattern.MatchString(normalizedID) {
			return fmt.Errorf("config: navivox.servers.%s id is invalid", id)
		}
		server.Bind = strings.TrimSpace(server.Bind)
		server.Profiles = normalizeNavivoxProfileIDs(server.Profiles)
		server.Transports = normalizeNavivoxStringSet(server.Transports)
		server.Capabilities = normalizeNavivoxStringSet(server.Capabilities)
		servers[normalizedID] = server
	}
	cfg.Servers = servers
	return nil
}

func normalizeNavivoxProfileIDs(values []string) []string {
	cleaned := compactStrings(values)
	if len(cleaned) == 0 {
		return nil
	}
	out := make([]string, 0, len(cleaned))
	seen := map[string]struct{}{}
	for _, value := range cleaned {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if !navivoxIDPattern.MatchString(value) {
			// Keep the server usable and let the route report degraded evidence
			// instead of failing unrelated profiles at config-load time.
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeNavivoxStringSet(values []string) []string {
	cleaned := compactStrings(values)
	if len(cleaned) == 0 {
		return nil
	}
	out := make([]string, 0, len(cleaned))
	seen := map[string]struct{}{}
	for _, value := range cleaned {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// NavivoxExposureRequiresVPN reports whether the given exposure_mode value
// requires bind_host to match an active VPN interface IP.
func NavivoxExposureRequiresVPN(mode string) bool {
	switch mode {
	case NavivoxExposureTailscale, NavivoxExposureWireGuard, NavivoxExposureVPN:
		return true
	default:
		return false
	}
}

// ValidateNavivoxBindAgainstVPN returns nil when navivox.bind_host either is
// not required to be a VPN interface IP (exposure_mode local/public, or
// channel disabled) or matches one of the live VPN IPs supplied by the
// caller. The list is supplied as plain strings so config has no dependency
// on the network/vpnhost package.
func ValidateNavivoxBindAgainstVPN(cfg *NavivoxCfg, vpnIPs []string) error {
	if cfg == nil || !cfg.Enabled {
		return nil
	}
	if !NavivoxExposureRequiresVPN(cfg.ExposureMode) {
		return nil
	}
	host := navivoxHostOnly(cfg.BindHost)
	if host == "" {
		return fmt.Errorf("config: navivox.bind_host is empty; exposure_mode=%s requires a VPN interface IP", cfg.ExposureMode)
	}
	for _, ip := range vpnIPs {
		if strings.EqualFold(strings.TrimSpace(ip), host) {
			return nil
		}
	}
	if len(vpnIPs) == 0 {
		return fmt.Errorf("config: navivox.exposure_mode=%s but no active VPN interface was detected; bind_host %q cannot be validated", cfg.ExposureMode, cfg.BindHost)
	}
	return fmt.Errorf("config: navivox.bind_host %q does not match any active VPN interface IP (%v); exposure_mode=%s requires a VPN bind", cfg.BindHost, vpnIPs, cfg.ExposureMode)
}

func navivoxValidateExposedToken(cfg *NavivoxCfg) error {
	if !navivoxExposureRequiresStrongToken(cfg) {
		return nil
	}
	if len(cfg.Token) < NavivoxMinExposedTokenLength {
		return fmt.Errorf("config: navivox.token must be at least %d characters when navivox.exposure_mode=%s", NavivoxMinExposedTokenLength, cfg.ExposureMode)
	}
	if !navivoxExposedTokenLooksRandom(cfg.Token) {
		return fmt.Errorf("config: navivox.token must include enough entropy for navivox.exposure_mode=%s; use a generated setup token", cfg.ExposureMode)
	}
	return nil
}

func navivoxExposedTokenLooksRandom(token string) bool {
	seen := map[rune]struct{}{}
	for _, r := range token {
		seen[r] = struct{}{}
	}
	return len(seen) >= NavivoxMinExposedTokenDistinctChars
}

func navivoxExposureRequiresStrongToken(cfg *NavivoxCfg) bool {
	if cfg == nil || cfg.ExposureMode == NavivoxExposureLocal {
		return false
	}
	switch cfg.AuthMode {
	case NavivoxAuthPairingToken, NavivoxAuthStaticToken, NavivoxAuthTokenAndTailscaleIdentity:
		return true
	default:
		return false
	}
}

func navivoxLoopbackHost(host string) bool {
	host = navivoxHostOnly(host)
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func navivoxWildcardHost(host string) bool {
	host = navivoxHostOnly(host)
	return host == "" || host == "0.0.0.0" || host == "::" || host == "[::]"
}

func navivoxWildcardOrigin(origins []string) bool {
	for _, origin := range origins {
		if strings.TrimSpace(origin) == "*" {
			return true
		}
	}
	return false
}

func navivoxHostOnly(raw string) string {
	host := strings.TrimSpace(raw)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return strings.Trim(strings.ToLower(host), "[]")
}
