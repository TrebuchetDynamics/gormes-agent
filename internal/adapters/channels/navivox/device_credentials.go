package navivox

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/navivox/capability"
)

// deviceCredentialRecord is the server-side record for one issued Navivox
// device credential. The raw secret is never stored: only its SHA-256 hash is
// retained so a future reconnect can validate a presented bearer without the
// gateway holding a recoverable secret. The credential is bound to one Gateway
// identity and one App install identity, per the durable-credential contract.
type deviceCredentialRecord struct {
	CredentialID string
	AppInstallID string
	GatewayID    string
	Scopes       []string
	CreatedAt    time.Time
	Revoked      bool
	secretHash   [32]byte
}

// navivoxDurableCredentialScopes caps the interim slice to a single coarse
// scope; finer scopes arrive with the keypair-challenge slice.
var navivoxDurableCredentialScopes = []string{"navivox"}

func (c *Channel) handleDeviceCredentials(w http.ResponseWriter, r *http.Request, _ string) {
	switch r.Method {
	case http.MethodPost:
		c.issueDeviceCredential(w, r)
	case http.MethodGet:
		c.listDeviceCredentials(w, r)
	default:
		writeNavivoxError(w, http.StatusMethodNotAllowed, "", "bad_request", "Method not allowed")
	}
}

func (c *Channel) issueDeviceCredential(w http.ResponseWriter, r *http.Request) {
	effectiveSecurity := navivoxTransportSecurityStatusForRequest(r, c.cfg).EffectiveSecurity
	if !capability.DurableReconnectSecurityAllowed(effectiveSecurity) {
		writeNavivoxError(w, http.StatusForbidden, "", "durable_reconnect_unavailable",
			"Durable reconnect is disabled on insecure non-loopback transport; use loopback, TLS, or an authenticated private network.")
		return
	}

	var body struct {
		AppInstallID string   `json:"app_install_id"`
		Scopes       []string `json:"scopes"`
	}
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err.Error() != "EOF" {
			writeNavivoxError(w, http.StatusBadRequest, "", "invalid_request", "Invalid JSON body")
			return
		}
	}
	appInstallID := strings.TrimSpace(body.AppInstallID)
	if appInstallID == "" {
		writeNavivoxError(w, http.StatusBadRequest, "", "invalid_request", "app_install_id is required")
		return
	}

	credentialID, err := navivoxRandomToken(16)
	if err != nil {
		writeNavivoxError(w, http.StatusInternalServerError, "", "internal_error", "Could not issue credential")
		return
	}
	secret, err := navivoxRandomToken(32)
	if err != nil {
		writeNavivoxError(w, http.StatusInternalServerError, "", "internal_error", "Could not issue credential")
		return
	}
	credentialID = "navivoxcred_" + credentialID
	secret = "nvbxdc_" + secret
	scopes := navivoxCappedDurableScopes(body.Scopes)
	now := c.now()

	record := &deviceCredentialRecord{
		CredentialID: credentialID,
		AppInstallID: appInstallID,
		GatewayID:    c.gatewayID,
		Scopes:       scopes,
		CreatedAt:    now,
		secretHash:   sha256.Sum256([]byte(secret)),
	}
	c.mu.Lock()
	c.deviceCredentials[credentialID] = record
	c.mu.Unlock()

	// The raw secret is returned exactly once and never logged or persisted.
	writeNavivoxJSON(w, http.StatusCreated, map[string]any{
		"object":         "gormes.navivox.device_credential",
		"credential_id":  credentialID,
		"secret":         secret,
		"auth_method":    "device_bearer",
		"interim":        true,
		"scopes":         scopes,
		"gateway_id":     c.gatewayID,
		"app_install_id": appInstallID,
		"created_at":     now.UTC().Format(time.RFC3339),
	})
	c.persistCredentialsToDisk()
}

func (c *Channel) listDeviceCredentials(w http.ResponseWriter, r *http.Request) {
	installFilter := strings.TrimSpace(r.URL.Query().Get("app_install_id"))

	c.mu.Lock()
	records := make([]*deviceCredentialRecord, 0, len(c.deviceCredentials))
	for _, record := range c.deviceCredentials {
		if installFilter != "" && record.AppInstallID != installFilter {
			continue
		}
		records = append(records, record)
	}
	c.mu.Unlock()

	sort.Slice(records, func(i, j int) bool {
		if records[i].CreatedAt.Equal(records[j].CreatedAt) {
			return records[i].CredentialID < records[j].CredentialID
		}
		return records[i].CreatedAt.Before(records[j].CreatedAt)
	})

	views := make([]map[string]any, 0, len(records))
	for _, record := range records {
		views = append(views, map[string]any{
			"credential_id":  record.CredentialID,
			"app_install_id": record.AppInstallID,
			"scopes":         record.Scopes,
			"created_at":     record.CreatedAt.UTC().Format(time.RFC3339),
			"revoked":        record.Revoked,
		})
	}
	// No secret or secret hash is ever included in list responses.
	writeNavivoxJSON(w, http.StatusOK, map[string]any{
		"object":      "gormes.navivox.device_credential_list",
		"gateway_id":  c.gatewayID,
		"credentials": views,
	})
}

func (c *Channel) handleDeviceCredentialRevoke(w http.ResponseWriter, r *http.Request, _ string) {
	if r.Method != http.MethodPost {
		writeNavivoxError(w, http.StatusMethodNotAllowed, "", "bad_request", "Method not allowed")
		return
	}
	var body struct {
		CredentialID string `json:"credential_id"`
	}
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err.Error() != "EOF" {
			writeNavivoxError(w, http.StatusBadRequest, "", "invalid_request", "Invalid JSON body")
			return
		}
	}
	credentialID := strings.TrimSpace(body.CredentialID)
	if credentialID == "" {
		writeNavivoxError(w, http.StatusBadRequest, "", "invalid_request", "credential_id is required")
		return
	}

	c.mu.Lock()
	if record, ok := c.deviceCredentials[credentialID]; ok {
		record.Revoked = true
	}
	c.mu.Unlock()

	// Idempotent: revocation reports success even for already-revoked or unknown
	// IDs so it never leaks which credential IDs exist.
	writeNavivoxJSON(w, http.StatusOK, map[string]any{
		"object":        "gormes.navivox.device_credential_revocation",
		"credential_id": credentialID,
		"revoked":       true,
	})
	c.persistCredentialsToDisk()
}

func navivoxCappedDurableScopes(requested []string) []string {
	allowed := map[string]bool{}
	for _, scope := range navivoxDurableCredentialScopes {
		allowed[scope] = true
	}
	out := make([]string, 0, len(requested))
	seen := map[string]bool{}
	for _, scope := range requested {
		trimmed := strings.TrimSpace(scope)
		if allowed[trimmed] && !seen[trimmed] {
			out = append(out, trimmed)
			seen[trimmed] = true
		}
	}
	if len(out) == 0 {
		return append([]string(nil), navivoxDurableCredentialScopes...)
	}
	return out
}

func navivoxRandomToken(nBytes int) (string, error) {
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
