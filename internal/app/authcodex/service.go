package authcodex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

const (
	OAuthIssuer   = "https://auth.openai.com"
	OAuthTokenURL = "https://auth.openai.com/oauth/token"
	OAuthClientID = "app_EMoamEEZ73f0CkXaXp7hrann"
	OAuthBaseURL  = "https://chatgpt.com/backend-api/codex"
)

type LoginRequest struct {
	Label    string
	Out      io.Writer
	Client   *http.Client
	Issuer   string
	TokenURL string
	ClientID string
	Timeout  time.Duration
	Sleep    func(time.Duration)
	Now      func() time.Time
}

func RunDeviceCodeLogin(ctx context.Context, req LoginRequest) (config.CodexOAuthTokens, error) {
	client := req.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	issuer := strings.TrimRight(firstNonEmpty(req.Issuer, OAuthIssuer), "/")
	tokenURL := firstNonEmpty(req.TokenURL, OAuthTokenURL)
	clientID := firstNonEmpty(req.ClientID, OAuthClientID)
	sleep := req.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	now := req.Now
	if now == nil {
		now = time.Now
	}
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Minute
	}

	device, err := RequestDeviceCode(ctx, client, issuer, clientID)
	if err != nil {
		return config.CodexOAuthTokens{}, err
	}
	if req.Out != nil {
		fmt.Fprintln(req.Out, "To continue, follow these steps:")
		fmt.Fprintln(req.Out)
		fmt.Fprintln(req.Out, "  1. Open this URL in your browser:")
		fmt.Fprintf(req.Out, "     %s/codex/device\n", issuer)
		fmt.Fprintln(req.Out)
		fmt.Fprintln(req.Out, "  2. Enter this code:")
		fmt.Fprintf(req.Out, "     %s\n", device.UserCode)
		fmt.Fprintln(req.Out)
		fmt.Fprintln(req.Out, "Waiting for sign-in... (press Ctrl+C to cancel)")
	}

	pollCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	code, err := PollDeviceCode(pollCtx, client, issuer, device, sleep)
	if err != nil {
		return config.CodexOAuthTokens{}, err
	}
	token, err := ExchangeDeviceCode(pollCtx, client, tokenURL, clientID, issuer+"/deviceauth/callback", code)
	if err != nil {
		return config.CodexOAuthTokens{}, err
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return config.CodexOAuthTokens{}, fmt.Errorf("token_exchange_no_access_token")
	}
	return config.CodexOAuthTokens{
		Label:        strings.TrimSpace(req.Label),
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		BaseURL:      strings.TrimRight(firstNonEmpty(os.Getenv("HERMES_CODEX_BASE_URL"), OAuthBaseURL), "/"),
		Source:       config.CodexOAuthSourceDeviceCode,
		LastRefresh:  now().UTC().Format(time.RFC3339),
	}, nil
}

type DeviceCode struct {
	UserCode     string
	DeviceAuthID string
	Interval     time.Duration
}

type AuthorizationCode struct {
	AuthorizationCode string
	CodeVerifier      string
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func RequestDeviceCode(ctx context.Context, client *http.Client, issuer, clientID string) (DeviceCode, error) {
	payload, _ := json.Marshal(map[string]string{"client_id": clientID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, issuer+"/api/accounts/deviceauth/usercode", bytes.NewReader(payload))
	if err != nil {
		return DeviceCode{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return DeviceCode{}, fmt.Errorf("device_code_request_failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return DeviceCode{}, fmt.Errorf("device_code_request_error")
	}
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return DeviceCode{}, fmt.Errorf("device_code_invalid_json")
	}
	device := DeviceCode{
		UserCode:     strings.TrimSpace(fmt.Sprint(raw["user_code"])),
		DeviceAuthID: strings.TrimSpace(fmt.Sprint(raw["device_auth_id"])),
		Interval:     time.Duration(maxInt(3, intFromAny(raw["interval"], 5))) * time.Second,
	}
	if device.UserCode == "" || device.DeviceAuthID == "" {
		return DeviceCode{}, fmt.Errorf("device_code_incomplete")
	}
	return device, nil
}

func PollDeviceCode(ctx context.Context, client *http.Client, issuer string, device DeviceCode, sleep func(time.Duration)) (AuthorizationCode, error) {
	for {
		select {
		case <-ctx.Done():
			return AuthorizationCode{}, fmt.Errorf("device_code_timeout")
		default:
		}
		sleep(device.Interval)
		payload, _ := json.Marshal(map[string]string{
			"device_auth_id": device.DeviceAuthID,
			"user_code":      device.UserCode,
		})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, issuer+"/api/accounts/deviceauth/token", bytes.NewReader(payload))
		if err != nil {
			return AuthorizationCode{}, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return AuthorizationCode{}, fmt.Errorf("device_code_poll_failed")
		}
		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			continue
		}
		if resp.StatusCode != http.StatusOK {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			return AuthorizationCode{}, fmt.Errorf("device_code_poll_error")
		}
		var raw struct {
			AuthorizationCode string `json:"authorization_code"`
			CodeVerifier      string `json:"code_verifier"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			_ = resp.Body.Close()
			return AuthorizationCode{}, fmt.Errorf("device_code_invalid_json")
		}
		_ = resp.Body.Close()
		code := AuthorizationCode{
			AuthorizationCode: strings.TrimSpace(raw.AuthorizationCode),
			CodeVerifier:      strings.TrimSpace(raw.CodeVerifier),
		}
		if code.AuthorizationCode == "" || code.CodeVerifier == "" {
			return AuthorizationCode{}, fmt.Errorf("device_code_incomplete_exchange")
		}
		return code, nil
	}
}

func ExchangeDeviceCode(ctx context.Context, client *http.Client, tokenURL, clientID, redirectURI string, code AuthorizationCode) (TokenResponse, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code.AuthorizationCode},
		"redirect_uri":  {redirectURI},
		"client_id":     {clientID},
		"code_verifier": {code.CodeVerifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return TokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("token_exchange_failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return TokenResponse{}, fmt.Errorf("token_exchange_error")
	}
	var token TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return TokenResponse{}, fmt.Errorf("token_exchange_invalid_json")
	}
	return token, nil
}

func SanitizeCommandError(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "auth_error"
	}
	lower := strings.ToLower(trimmed)
	for _, marker := range []string{"access_token", "refresh_token", "authorization", "bearer ", "client_secret"} {
		if strings.Contains(lower, marker) {
			return "[redacted]"
		}
	}
	if len(trimmed) > 160 {
		return trimmed[:160]
	}
	return trimmed
}

func intFromAny(value any, fallback int) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return fallback
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
