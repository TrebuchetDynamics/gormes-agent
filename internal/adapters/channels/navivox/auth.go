package navivox

import (
	"crypto/hmac"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/channelutil"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

type authenticatedHandler func(http.ResponseWriter, *http.Request, string)

func (c *Channel) withAuth(next authenticatedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, ok := c.authenticate(r)
		if !ok {
			writeNavivoxError(w, http.StatusUnauthorized, "", "unauthorized", "Unauthorized")
			return
		}
		next(w, r, identity)
	}
}

func (c *Channel) authenticate(r *http.Request) (string, bool) {
	mode := strings.ToLower(strings.TrimSpace(c.cfg.AuthMode))
	switch mode {
	case config.NavivoxAuthTailscaleIdentity:
		return c.authenticateTailscaleIdentity(r)
	case config.NavivoxAuthPairingToken, config.NavivoxAuthStaticToken:
		if c.authenticateToken(r) {
			return "token", true
		}
		return "", false
	case config.NavivoxAuthTokenAndTailscaleIdentity:
		identity, ok := c.authenticateTailscaleIdentity(r)
		if !ok || !c.authenticateToken(r) {
			return "", false
		}
		return identity, true
	default:
		return "", false
	}
}

func (c *Channel) authenticateTailscaleIdentity(r *http.Request) (string, bool) {
	identity := firstHeader(r, "Tailscale-User-Login", "X-Tailscale-User-Login", "Tailscale-Device-Name", "X-Tailscale-Device-Name")
	if identity == "" {
		return "", false
	}
	if len(c.cfg.AllowedTailnetIdentities) == 0 {
		return identity, true
	}
	if channelutil.ContainsEqualFold(c.cfg.AllowedTailnetIdentities, identity) {
		return identity, true
	}
	return "", false
}

func (c *Channel) authenticateToken(r *http.Request) bool {
	token := bearerToken(r)
	if token == "" {
		token = strings.TrimSpace(r.Header.Get("X-Gormes-Navivox-Token"))
	}
	if token == "" {
		token = webSocketProtocolToken(r)
	}
	if token == "" || c.cfg.Token == "" {
		return false
	}
	return hmac.Equal([]byte(token), []byte(c.cfg.Token))
}

func bearerToken(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
}

func webSocketProtocolToken(r *http.Request) string {
	for _, protocol := range websocket.Subprotocols(r) {
		if !strings.HasPrefix(protocol, navivoxWebSocketTokenProtocolPrefix) {
			continue
		}
		encoded := strings.TrimPrefix(protocol, navivoxWebSocketTokenProtocolPrefix)
		decoded, err := base64.RawURLEncoding.DecodeString(encoded)
		if err != nil {
			return ""
		}
		return string(decoded)
	}
	return ""
}

func firstHeader(r *http.Request, names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(r.Header.Get(name)); value != "" {
			return value
		}
	}
	return ""
}

func (c *Channel) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" && c.originAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Gormes-Navivox-Token")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (c *Channel) originAllowed(origin string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return true
	}
	return channelutil.ContainsString(c.cfg.AllowOrigins, "*") || channelutil.ContainsEqualFold(c.cfg.AllowOrigins, origin)
}
