package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// NousOAuthCredentials is the local type for OAuth state, mirroring
// config.NousOAuthCredentials to avoid an import cycle.
type NousOAuthCredentials struct {
	Label              string
	PortalBaseURL      string
	InferenceBaseURL   string
	ClientID           string
	Scope              string
	TokenType          string
	AccessToken        string
	RefreshToken       string
	ObtainedAt         string
	ExpiresAt          string
	ExpiresIn          int
	AgentKey           string
	AgentKeyID         string
	AgentKeyExpiresAt  string
	AgentKeyExpiresIn  int
	AgentKeyObtainedAt string
}

// CredentialPoolOptions is a local stub for config.CredentialPoolOptions.
type CredentialPoolOptions struct {
	HermesHome string
}

// NousOAuthLoginOptions controls the device-code login flow.
// All fields except PortalBaseURL are optional; defaults come from
// the upstream Hermes PROVIDER_REGISTRY["nous"] config.
type NousOAuthLoginOptions struct {
	PortalBaseURL    string
	InferenceBaseURL string
	ClientID         string
	Scope            string
	HTTPClient       *http.Client
	OpenBrowser      func(verificationURL string) error
}

// NousOAuthRefreshOptions controls refresh-token rotation.
type NousOAuthRefreshOptions struct {
	PortalBaseURL string
	ClientID      string
	RefreshToken  string
	HTTPClient    *http.Client
}

// NousOAuthMintOptions controls short-lived agent-key minting.
type NousOAuthMintOptions struct {
	PortalBaseURL    string
	AccessToken      string
	MinKeyTTLSeconds int
	HTTPClient       *http.Client
}

// NousOAuthRuntimeOptions controls runtime credential resolution.
type NousOAuthRuntimeOptions struct {
	MinKeyTTLSeconds int
	HTTPClient       *http.Client
}

// NousRuntimeCredentials is the resolved inference credential ready
// for provider use.
type NousRuntimeCredentials struct {
	Provider  string `json:"provider"`
	BaseURL   string `json:"base_url"`
	APIKey    string `json:"api_key"`
	KeyID     string `json:"key_id"`
	ExpiresAt string `json:"expires_at"`
	Source    string `json:"source"`
}

// NousAuthError is a classified OAuth error with actionable operator guidance.
type NousAuthError struct {
	Message         string
	Code            string
	ReloginRequired bool
}

func (e *NousAuthError) Error() string {
	return fmt.Sprintf("nous oauth: %s (code=%s)", e.Message, e.Code)
}

func httpClientOrDefault(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func parseNousError(prefix string, resp *http.Response) *NousAuthError {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	var payload struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	_ = json.Unmarshal(body, &payload)
	desc := payload.ErrorDescription
	if desc == "" {
		desc = payload.Error
	}
	if desc == "" {
		desc = fmt.Sprintf("%s returned status %d", prefix, resp.StatusCode)
	}
	code, relogin := classifyNousOAuthError(prefix, payload.Error, desc)
	return &NousAuthError{
		Message:         desc,
		Code:            code,
		ReloginRequired: relogin,
	}
}

func classifyNousOAuthError(operation, errCode, description string) (code string, relogin bool) {
	lowered := strings.ToLower(description)
	if strings.Contains(operation, "agent key mint") {
		return "agent_key_minting_failed", false
	}
	if strings.Contains(operation, "device code") {
		return "device_code_expired", true
	}
	if strings.Contains(lowered, "reuse") || strings.Contains(lowered, "reuse detected") {
		return "refresh_token_revoked", true
	}
	switch errCode {
	case "expired_token":
		return "device_code_expired", true
	case "invalid_grant", "invalid_token":
		return "refresh_token_revoked", true
	case "access_denied":
		return "device_code_expired", true
	default:
		if strings.Contains(lowered, "expired") {
			return "device_code_expired", true
		}
		return "agent_key_minting_failed", true
	}
}

func doNousJSON(ctx context.Context, client *http.Client, method, urlStr string, body url.Values, headers map[string]string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = strings.NewReader(body.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, urlStr, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("nous oauth: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return client.Do(req)
}

// NousOAuthDeviceCodeLogin runs the full Hermes-compatible Nous device-code
// OAuth flow: request device code, open browser for user approval, poll for
// token, then immediately mint a short-lived agent key. Returns full
// credentials suitable for SaveNousOAuthCredentials.
func NousOAuthDeviceCodeLogin(ctx context.Context, opts NousOAuthLoginOptions) (NousOAuthCredentials, error) {
	client := httpClientOrDefault(opts.HTTPClient)
	portalBase := strings.TrimRight(opts.PortalBaseURL, "/")
	inferenceBase := strings.TrimRight(opts.InferenceBaseURL, "/")
	clientID := opts.ClientID
	scope := opts.Scope

	resp, err := doNousJSON(ctx, client, http.MethodPost,
		portalBase+"/api/oauth/device/code",
		url.Values{"client_id": {clientID}, "scope": {scope}}, nil)
	if err != nil {
		return NousOAuthCredentials{}, fmt.Errorf("nous oauth: device code request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return NousOAuthCredentials{}, parseNousError("device code request", resp)
	}

	var deviceResp struct {
		DeviceCode              string `json:"device_code"`
		UserCode                string `json:"user_code"`
		VerificationURIComplete string `json:"verification_uri_complete"`
		ExpiresIn               int    `json:"expires_in"`
		Interval                int    `json:"interval"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&deviceResp); err != nil {
		return NousOAuthCredentials{}, fmt.Errorf("nous oauth: decode device code response: %w", err)
	}

	if opts.OpenBrowser != nil && deviceResp.VerificationURIComplete != "" {
		_ = opts.OpenBrowser(deviceResp.VerificationURIComplete)
	}

	pollInterval := deviceResp.Interval
	if pollInterval < 1 {
		pollInterval = 5
	}
	if pollInterval > 10 {
		pollInterval = 10
	}

	deadline := time.Now().Add(time.Duration(deviceResp.ExpiresIn) * time.Second)
	ticker := time.NewTicker(time.Duration(pollInterval) * time.Second)
	defer ticker.Stop()

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
	}

	pollDone := false
	for !pollDone {
		select {
		case <-ctx.Done():
			return NousOAuthCredentials{}, ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return NousOAuthCredentials{}, &NousAuthError{
					Message:         "device code expired before approval",
					Code:            "device_code_expired",
					ReloginRequired: true,
				}
			}
			tokenResp, err = pollDeviceToken(ctx, client, portalBase, clientID, deviceResp.DeviceCode)
			if err == nil {
				pollDone = true
				break
			}
			authErr, ok := err.(*NousAuthError)
			if ok && authErr.Code == "device_code_expired" {
				return NousOAuthCredentials{}, authErr
			}
		}
	}

	now := time.Now().UTC()
	creds := NousOAuthCredentials{
		PortalBaseURL:    portalBase,
		InferenceBaseURL: inferenceBase,
		ClientID:         clientID,
		Scope:            tokenResp.Scope,
		TokenType:        tokenResp.TokenType,
		AccessToken:      tokenResp.AccessToken,
		RefreshToken:     tokenResp.RefreshToken,
		ObtainedAt:       now.Format(time.RFC3339),
		ExpiresIn:        tokenResp.ExpiresIn,
		ExpiresAt:        now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Format(time.RFC3339),
	}

	minted, err := MintNousAgentKey(ctx, NousOAuthMintOptions{
		PortalBaseURL:    portalBase,
		AccessToken:      creds.AccessToken,
		MinKeyTTLSeconds: 300,
		HTTPClient:       client,
	})
	if err != nil {
		return creds, fmt.Errorf("nous oauth: device code succeeded but agent key mint failed: %w", err)
	}

	creds.AgentKey = minted.AgentKey
	creds.AgentKeyID = minted.AgentKeyID
	creds.AgentKeyExpiresAt = minted.AgentKeyExpiresAt
	creds.AgentKeyObtainedAt = now.Format(time.RFC3339)
	creds.AgentKeyExpiresIn = 86400

	return creds, nil
}

func pollDeviceToken(ctx context.Context, client *http.Client, portalBase, clientID, deviceCode string) (struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}, error) {
	var zero struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
	}
	resp, err := doNousJSON(ctx, client, http.MethodPost,
		portalBase+"/api/oauth/token",
		url.Values{
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"device_code": {deviceCode},
			"client_id":   {clientID},
		}, nil)
	if err != nil {
		return zero, fmt.Errorf("nous oauth: poll token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return zero, parseNousError("poll token", resp)
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return zero, fmt.Errorf("nous oauth: decode poll response: %w", err)
	}
	if result.AccessToken == "" {
		return zero, &NousAuthError{
			Message:         "poll response missing access_token",
			Code:            "invalid_token",
			ReloginRequired: true,
		}
	}
	return result, nil
}

// RefreshNousAccessToken exchanges a refresh token for a new
// access-token + refresh-token pair via the X-Nous-Refresh-Token header.
// Returns an updated NousOAuthCredentials with new tokens.
func RefreshNousAccessToken(ctx context.Context, opts NousOAuthRefreshOptions) (NousOAuthCredentials, error) {
	client := httpClientOrDefault(opts.HTTPClient)
	portalBase := strings.TrimRight(opts.PortalBaseURL, "/")

	resp, err := doNousJSON(ctx, client, http.MethodPost,
		portalBase+"/api/oauth/token",
		url.Values{
			"grant_type": {"refresh_token"},
			"client_id":  {opts.ClientID},
		},
		map[string]string{"x-nous-refresh-token": opts.RefreshToken},
	)
	if err != nil {
		return NousOAuthCredentials{}, fmt.Errorf("nous oauth: refresh token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return NousOAuthCredentials{}, parseNousError("refresh token", resp)
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return NousOAuthCredentials{}, fmt.Errorf("nous oauth: decode refresh response: %w", err)
	}
	if result.AccessToken == "" {
		return NousOAuthCredentials{}, &NousAuthError{
			Message:         "refresh response missing access_token",
			Code:            "invalid_token",
			ReloginRequired: true,
		}
	}

	now := time.Now().UTC()
	return NousOAuthCredentials{
		PortalBaseURL: opts.PortalBaseURL,
		AccessToken:   result.AccessToken,
		RefreshToken:  result.RefreshToken,
		TokenType:     result.TokenType,
		ExpiresIn:     result.ExpiresIn,
		ExpiresAt:     now.Add(time.Duration(result.ExpiresIn) * time.Second).Format(time.RFC3339),
		ObtainedAt:    now.Format(time.RFC3339),
	}, nil
}

// MintNousAgentKey obtains a short-lived (24h) inference API key from
// the Nous portal using a valid access token. Returns credentials with
// the agent key populated.
func MintNousAgentKey(ctx context.Context, opts NousOAuthMintOptions) (NousOAuthCredentials, error) {
	client := httpClientOrDefault(opts.HTTPClient)
	portalBase := strings.TrimRight(opts.PortalBaseURL, "/")
	minTTL := opts.MinKeyTTLSeconds
	if minTTL < 60 {
		minTTL = 60
	}

	var mintReq struct {
		MinTTLSeconds int `json:"min_ttl_seconds"`
	}
	mintReq.MinTTLSeconds = minTTL
	reqBody, _ := json.Marshal(mintReq)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		portalBase+"/api/oauth/agent-key",
		strings.NewReader(string(reqBody)))
	if err != nil {
		return NousOAuthCredentials{}, fmt.Errorf("nous oauth: mint agent key: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+opts.AccessToken)

	resp, err := client.Do(req)
	if err != nil {
		return NousOAuthCredentials{}, fmt.Errorf("nous oauth: mint agent key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return NousOAuthCredentials{}, parseNousError("agent key mint", resp)
	}

	var result struct {
		APIKey    string `json:"api_key"`
		KeyID     string `json:"key_id"`
		ExpiresAt string `json:"expires_at"`
		ExpiresIn int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return NousOAuthCredentials{}, fmt.Errorf("nous oauth: decode agent key response: %w", err)
	}
	if result.APIKey == "" {
		return NousOAuthCredentials{}, &NousAuthError{
			Message:         "mint response missing api_key",
			Code:            "agent_key_minting_failed",
			ReloginRequired: false,
		}
	}

	now := time.Now().UTC()
	return NousOAuthCredentials{
		PortalBaseURL:      opts.PortalBaseURL,
		AgentKey:           result.APIKey,
		AgentKeyID:         result.KeyID,
		AgentKeyExpiresAt:  result.ExpiresAt,
		AgentKeyExpiresIn:  result.ExpiresIn,
		AgentKeyObtainedAt: now.Format(time.RFC3339),
	}, nil
}

// ResolveNousRuntimeCredentials resolves inference credentials from
// previously-loaded Nous OAuth state. It refreshes the access token if
// needed, mints a fresh agent key if the cached one is near expiry, and
// returns inference credentials ready for provider use.
// Callers (cmd/gormes/auth.go) load credentials via config.LoadNousOAuthCredentials
// and pass them here.
func ResolveNousRuntimeCredentials(ctx context.Context, creds NousOAuthCredentials, opts NousOAuthRuntimeOptions) (NousRuntimeCredentials, error) {
	return resolveNousRuntimeFromCreds(ctx, creds, opts)
}

func resolveNousRuntimeFromCreds(ctx context.Context, creds NousOAuthCredentials, opts NousOAuthRuntimeOptions) (NousRuntimeCredentials, error) {
	client := httpClientOrDefault(opts.HTTPClient)
	minKeyTTL := opts.MinKeyTTLSeconds
	if minKeyTTL < 60 {
		minKeyTTL = 60
	}

	now := time.Now().UTC()

	// Check if access token needs refresh
	accessTokenExpired := false
	if creds.ExpiresAt != "" {
		expAt, err := time.Parse(time.RFC3339, creds.ExpiresAt)
		if err == nil && now.Add(time.Duration(minKeyTTL)*time.Second).After(expAt) {
			accessTokenExpired = true
		}
	}
	if creds.AccessToken == "" {
		accessTokenExpired = true
	}

	if accessTokenExpired {
		if creds.RefreshToken == "" {
			return NousRuntimeCredentials{}, &NousAuthError{
				Message:         "access token expired and no refresh token available",
				Code:            "refresh_token_missing",
				ReloginRequired: true,
			}
		}
		refreshed, err := RefreshNousAccessToken(ctx, NousOAuthRefreshOptions{
			PortalBaseURL: creds.PortalBaseURL,
			ClientID:      creds.ClientID,
			RefreshToken:  creds.RefreshToken,
			HTTPClient:    client,
		})
		if err != nil {
			return NousRuntimeCredentials{}, err
		}
		creds.AccessToken = refreshed.AccessToken
		creds.RefreshToken = refreshed.RefreshToken
		creds.ExpiresAt = refreshed.ExpiresAt
		creds.ExpiresIn = refreshed.ExpiresIn
	}

	// Check if agent key needs minting
	agentKeyValid := false
	if creds.AgentKey != "" && creds.AgentKeyExpiresAt != "" {
		expAt, err := time.Parse(time.RFC3339, creds.AgentKeyExpiresAt)
		if err == nil && now.Add(time.Duration(minKeyTTL)*time.Second).Before(expAt) {
			agentKeyValid = true
		}
	}

	if agentKeyValid {
		return NousRuntimeCredentials{
			Provider:  "nous",
			BaseURL:   creds.InferenceBaseURL,
			APIKey:    creds.AgentKey,
			KeyID:     creds.AgentKeyID,
			ExpiresAt: creds.AgentKeyExpiresAt,
			Source:    "cache",
		}, nil
	}

	// Mint fresh agent key
	minted, err := MintNousAgentKey(ctx, NousOAuthMintOptions{
		PortalBaseURL:    creds.PortalBaseURL,
		AccessToken:      creds.AccessToken,
		MinKeyTTLSeconds: minKeyTTL,
		HTTPClient:       client,
	})
	if err != nil {
		return NousRuntimeCredentials{}, err
	}

	return NousRuntimeCredentials{
		Provider:  "nous",
		BaseURL:   creds.InferenceBaseURL,
		APIKey:    minted.AgentKey,
		KeyID:     minted.AgentKeyID,
		ExpiresAt: minted.AgentKeyExpiresAt,
		Source:    "portal",
	}, nil
}
