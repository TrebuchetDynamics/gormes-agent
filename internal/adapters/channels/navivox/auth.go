package navivox

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/channelutil"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

const (
	navivoxAuthFailureLimit  = 5
	navivoxAuthFailureWindow = time.Minute
)

type authenticatedHandler func(http.ResponseWriter, *http.Request, string)

type navivoxAuthFailureState struct {
	Count       int
	FirstAt     time.Time
	LockedUntil time.Time
}

func (c *Channel) withAuth(next authenticatedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c.authRateLimited(r) {
			writeNavivoxError(w, http.StatusTooManyRequests, "", "auth_rate_limited", "Authentication attempts are temporarily rate limited")
			return
		}
		if navivoxRequestHasURLCredential(r) {
			c.recordAuthFailure(r)
			writeNavivoxError(w, http.StatusUnauthorized, "", "url_credentials_rejected", "URL credentials are not accepted")
			return
		}
		identity, ok := c.authenticate(r)
		if !ok {
			c.recordAuthFailure(r)
			writeNavivoxError(w, http.StatusUnauthorized, "", "unauthorized", "Unauthorized")
			return
		}
		c.clearAuthFailures(r)
		next(w, r, identity)
	}
}

func (c *Channel) authRateLimited(r *http.Request) bool {
	key := navivoxAuthFailureKey(r)
	if key == "" {
		return false
	}
	now := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()
	state, ok := c.authFailures[key]
	if !ok {
		return false
	}
	if !state.LockedUntil.IsZero() {
		if now.Before(state.LockedUntil) {
			return true
		}
		delete(c.authFailures, key)
		return false
	}
	if !state.FirstAt.IsZero() && now.Sub(state.FirstAt) >= navivoxAuthFailureWindow {
		delete(c.authFailures, key)
	}
	return false
}

func (c *Channel) recordAuthFailure(r *http.Request) {
	key := navivoxAuthFailureKey(r)
	if key == "" {
		return
	}
	now := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.authFailures == nil {
		c.authFailures = map[string]navivoxAuthFailureState{}
	}
	state := c.authFailures[key]
	if state.FirstAt.IsZero() || now.Sub(state.FirstAt) >= navivoxAuthFailureWindow || (!state.LockedUntil.IsZero() && !now.Before(state.LockedUntil)) {
		state = navivoxAuthFailureState{FirstAt: now}
	}
	state.Count++
	if state.Count >= navivoxAuthFailureLimit {
		state.LockedUntil = now.Add(navivoxAuthFailureWindow)
	}
	c.authFailures[key] = state
}

func (c *Channel) clearAuthFailures(r *http.Request) {
	key := navivoxAuthFailureKey(r)
	if key == "" {
		return
	}
	c.mu.Lock()
	delete(c.authFailures, key)
	c.mu.Unlock()
}

func navivoxAuthFailureKey(r *http.Request) string {
	if r == nil {
		return ""
	}
	remote := strings.TrimSpace(r.RemoteAddr)
	if remote == "" {
		return "unknown"
	}
	if host, _, err := net.SplitHostPort(remote); err == nil && strings.TrimSpace(host) != "" {
		return strings.TrimSpace(host)
	}
	return remote
}

func (c *Channel) authenticate(r *http.Request) (string, bool) {
	// Device bearer supersedes the channel's bootstrap auth mode: any client
	// that was previously issued a device credential can reconnect without
	// re-scanning the QR code, regardless of how the channel was initially
	// configured (pairing_token, static_token, etc.).
	if identity, ok := c.authenticateDeviceBearer(r); ok {
		return identity, true
	}
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
	case config.NavivoxAuthDeviceBearer:
		// Pure device-bearer mode (for serve with no bootstrap token).
		return "", false
	default:
		return "", false
	}
}

// authenticateDeviceBearer validates a device credential bearer token of the
// form "{credentialId}:{secret}". The secret is never stored; only its
// SHA-256 hash is compared against the record kept at credential-issuance time.
func (c *Channel) authenticateDeviceBearer(r *http.Request) (string, bool) {
	token, ok := navivoxSingleTokenCredential(r)
	if !ok || token == "" {
		return "", false
	}
	sep := strings.IndexByte(token, ':')
	if sep <= 0 || sep >= len(token)-1 {
		return "", false
	}
	credentialID := token[:sep]
	secret := token[sep+1:]
	if strings.TrimSpace(credentialID) == "" || strings.TrimSpace(secret) == "" {
		return "", false
	}
	c.mu.Lock()
	record, ok := c.deviceCredentials[credentialID]
	c.mu.Unlock()
	if !ok || record == nil || record.Revoked {
		return "", false
	}
	hash := sha256.Sum256([]byte(secret))
	if !hmac.Equal(hash[:], record.secretHash[:]) {
		return "", false
	}
	return credentialID, true
}

func (c *Channel) authenticateTailscaleIdentity(r *http.Request) (string, bool) {
	identity, present, ok := singleHeaderAliasValue(r.Header, "Tailscale-User-Login", "X-Tailscale-User-Login")
	if !ok {
		return "", false
	}
	if !present {
		identity, present, ok = singleHeaderAliasValue(r.Header, "Tailscale-Device-Name", "X-Tailscale-Device-Name")
		if !ok {
			return "", false
		}
	}
	if !present || identity == "" {
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
	token, ok := navivoxSingleTokenCredential(r)
	if !ok || token == "" || c.cfg.Token == "" {
		return false
	}
	return hmac.Equal([]byte(token), []byte(c.cfg.Token))
}

func navivoxSingleTokenCredential(r *http.Request) (string, bool) {
	var tokens []string
	authorizationValues := nonEmptyHeaderValues(r.Header, "Authorization")
	if len(authorizationValues) > 1 {
		return "", false
	}
	if len(authorizationValues) == 1 {
		tokens = append(tokens, bearerTokenValue(authorizationValues[0]))
	}
	navivoxTokenValues := nonEmptyHeaderValues(r.Header, "X-Gormes-Navivox-Token")
	if len(navivoxTokenValues) > 1 {
		return "", false
	}
	if len(navivoxTokenValues) == 1 {
		tokens = append(tokens, navivoxTokenValues[0])
	}
	if token, present := webSocketProtocolToken(r); present {
		tokens = append(tokens, token)
	}
	if len(tokens) != 1 || strings.TrimSpace(tokens[0]) == "" {
		return "", false
	}
	return tokens[0], true
}

func nonEmptyHeaderValues(header http.Header, name string) []string {
	var values []string
	for _, value := range header.Values(name) {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func navivoxRequestHasURLCredential(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	for name := range r.URL.Query() {
		normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), "-", "_")
		switch normalized {
		case "token", "access_token", "auth_token", "rest_token", "navivox_token", "gormes_navivox_token":
			return true
		}
	}
	return false
}

func bearerToken(r *http.Request) string {
	values := nonEmptyHeaderValues(r.Header, "Authorization")
	if len(values) != 1 {
		return ""
	}
	return bearerTokenValue(values[0])
}

func bearerTokenValue(value string) string {
	auth := strings.TrimSpace(value)
	separator := strings.IndexByte(auth, ' ')
	if separator <= 0 || !strings.EqualFold(auth[:separator], "Bearer") {
		return ""
	}
	return strings.TrimSpace(auth[separator+1:])
}

func webSocketProtocolToken(r *http.Request) (string, bool) {
	found := false
	var token string
	for _, protocol := range websocket.Subprotocols(r) {
		if !strings.HasPrefix(protocol, navivoxWebSocketTokenProtocolPrefix) {
			continue
		}
		if found {
			return "", true
		}
		found = true
		encoded := strings.TrimPrefix(protocol, navivoxWebSocketTokenProtocolPrefix)
		decoded, err := base64.RawURLEncoding.DecodeString(encoded)
		if err != nil {
			return "", true
		}
		token = string(decoded)
	}
	return token, found
}

func singleHeaderAliasValue(header http.Header, names ...string) (value string, present bool, ok bool) {
	for _, name := range names {
		for _, candidate := range nonEmptyHeaderValues(header, name) {
			if !present {
				value = candidate
				present = true
				continue
			}
			if !strings.EqualFold(value, candidate) {
				return "", true, false
			}
		}
	}
	return value, present, true
}

func (c *Channel) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" {
			if !c.originAllowed(origin) {
				writeNavivoxError(w, http.StatusForbidden, "", "forbidden_origin", "Origin is not allowed")
				return
			}
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
	if channelutil.ContainsString(c.cfg.AllowOrigins, "*") || channelutil.ContainsEqualFold(c.cfg.AllowOrigins, origin) {
		return true
	}
	return c.cfg.ExposureMode == config.NavivoxExposureLocal && navivoxLoopbackBrowserOrigin(origin)
}

func navivoxLoopbackBrowserOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	switch parsed.Scheme {
	case "http", "https":
	default:
		return false
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Path != "" {
		return false
	}
	host := parsed.Hostname()
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
