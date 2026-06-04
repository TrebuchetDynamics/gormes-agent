package setupnavivox

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/network/vpnhost"
)

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

// CSV trims a comma-separated setup answer into non-empty values.
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
