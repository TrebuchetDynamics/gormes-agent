package navivoxconnect

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// ServerBindHostPort resolves an optional server bind address against the
// Navivox default bind host and port.
func ServerBindHostPort(bind, defaultHost string, defaultPort int) (string, int) {
	bind = strings.TrimSpace(bind)
	if bind == "" {
		return defaultHost, defaultPort
	}
	if host, portText, err := net.SplitHostPort(bind); err == nil {
		if port, parseErr := strconv.Atoi(portText); parseErr == nil && port > 0 {
			return host, port
		}
		return host, defaultPort
	}
	return strings.Trim(bind, "[]"), defaultPort
}

// URLs returns the HTTP base URL and WebSocket stream URL for a Navivox host.
func URLs(host string, port int) (baseURL, webSocketURL string) {
	host = strings.Trim(strings.TrimSpace(host), "[]")
	hostPort := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	baseURL = "http://" + hostPort
	webSocketURL = "ws://" + hostPort + "/v1/navivox/stream"
	return baseURL, webSocketURL
}
