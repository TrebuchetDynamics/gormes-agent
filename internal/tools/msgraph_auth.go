package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	DefaultMicrosoftGraphScope        = "https://graph.microsoft.com/.default"
	DefaultMicrosoftGraphAuthorityURL = "https://login.microsoftonline.com"

	MicrosoftGraphNotConfigured      = "msgraph_not_configured"
	MicrosoftGraphTokenUnavailable   = "msgraph_token_unavailable"
	MicrosoftGraphRequestUnavailable = "msgraph_request_unavailable"

	defaultMicrosoftGraphTokenSkew = 120 * time.Second
)

var microsoftGraphBearerPattern = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+`)

type MicrosoftGraphHTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type MicrosoftGraphError struct {
	Evidence string
	Message  string
}

func (e *MicrosoftGraphError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message == "" {
		return e.Evidence
	}
	return e.Evidence + ": " + e.Message
}

func AsMicrosoftGraphError(err error, target **MicrosoftGraphError) bool {
	return errors.As(err, target)
}

type MicrosoftGraphCredentials struct {
	TenantID     string
	ClientID     string
	ClientSecret string
	Scope        string
	AuthorityURL string
}

func MicrosoftGraphCredentialsFromEnv(env map[string]string, required bool) (*MicrosoftGraphCredentials, error) {
	lookup := func(key string) string {
		if env == nil {
			return os.Getenv(key)
		}
		return env[key]
	}
	creds := MicrosoftGraphCredentials{
		TenantID:     strings.TrimSpace(lookup("MSGRAPH_TENANT_ID")),
		ClientID:     strings.TrimSpace(lookup("MSGRAPH_CLIENT_ID")),
		ClientSecret: strings.TrimSpace(lookup("MSGRAPH_CLIENT_SECRET")),
		Scope:        strings.TrimSpace(lookup("MSGRAPH_SCOPE")),
		AuthorityURL: strings.TrimSpace(lookup("MSGRAPH_AUTHORITY_URL")),
	}
	if creds.Scope == "" {
		creds.Scope = DefaultMicrosoftGraphScope
	}
	if creds.AuthorityURL == "" {
		creds.AuthorityURL = DefaultMicrosoftGraphAuthorityURL
	}
	missing := make([]string, 0, 3)
	if creds.TenantID == "" {
		missing = append(missing, "MSGRAPH_TENANT_ID")
	}
	if creds.ClientID == "" {
		missing = append(missing, "MSGRAPH_CLIENT_ID")
	}
	if creds.ClientSecret == "" {
		missing = append(missing, "MSGRAPH_CLIENT_SECRET")
	}
	if len(missing) > 0 {
		if !required {
			return nil, nil
		}
		return nil, &MicrosoftGraphError{
			Evidence: MicrosoftGraphNotConfigured,
			Message:  "missing Microsoft Graph configuration: " + strings.Join(missing, ", "),
		}
	}
	return &creds, nil
}

func (c MicrosoftGraphCredentials) TokenURL() string {
	base := strings.TrimRight(strings.TrimSpace(c.AuthorityURL), "/")
	if base == "" {
		base = DefaultMicrosoftGraphAuthorityURL
	}
	tenant := strings.Trim(strings.TrimSpace(c.TenantID), "/")
	return base + "/" + url.PathEscape(tenant) + "/oauth2/v2.0/token"
}

type MicrosoftGraphAccessToken struct {
	AccessToken string
	TokenType   string
	ExpiresAt   time.Time
}

func (t MicrosoftGraphAccessToken) expired(now time.Time, skew time.Duration) bool {
	if t.AccessToken == "" || t.ExpiresAt.IsZero() {
		return true
	}
	if skew < 0 {
		skew = 0
	}
	return !t.ExpiresAt.After(now.Add(skew))
}

type MicrosoftGraphTokenProviderOptions struct {
	HTTPClient  MicrosoftGraphHTTPClient
	Skew        time.Duration
	Now         func() time.Time
	RedactExtra []string
}

type MicrosoftGraphTokenProvider struct {
	credentials MicrosoftGraphCredentials
	client      MicrosoftGraphHTTPClient
	skew        time.Duration
	now         func() time.Time
	redact      []string

	mu     sync.Mutex
	cached *MicrosoftGraphAccessToken
}

func NewMicrosoftGraphTokenProvider(creds MicrosoftGraphCredentials, opts MicrosoftGraphTokenProviderOptions) *MicrosoftGraphTokenProvider {
	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	skew := opts.Skew
	if skew == 0 {
		skew = defaultMicrosoftGraphTokenSkew
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	redact := append([]string{creds.ClientSecret}, opts.RedactExtra...)
	return &MicrosoftGraphTokenProvider{
		credentials: normalizeMicrosoftGraphCredentials(creds),
		client:      client,
		skew:        skew,
		now:         now,
		redact:      redact,
	}
}

func (p *MicrosoftGraphTokenProvider) GetAccessToken(ctx context.Context, forceRefresh bool) (string, error) {
	if p == nil {
		return "", &MicrosoftGraphError{Evidence: MicrosoftGraphTokenUnavailable, Message: "Microsoft Graph token provider is nil"}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !forceRefresh && p.cached != nil && !p.cached.expired(p.now(), p.skew) {
		return p.cached.AccessToken, nil
	}
	token, err := p.fetchAccessToken(ctx)
	if err != nil {
		return "", err
	}
	p.cached = &token
	return token.AccessToken, nil
}

func (p *MicrosoftGraphTokenProvider) ClearCache() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cached = nil
}

func (p *MicrosoftGraphTokenProvider) fetchAccessToken(ctx context.Context) (MicrosoftGraphAccessToken, error) {
	values := url.Values{}
	values.Set("grant_type", "client_credentials")
	values.Set("client_id", p.credentials.ClientID)
	values.Set("client_secret", p.credentials.ClientSecret)
	values.Set("scope", p.credentials.Scope)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.credentials.TokenURL(), strings.NewReader(values.Encode()))
	if err != nil {
		return MicrosoftGraphAccessToken{}, &MicrosoftGraphError{Evidence: MicrosoftGraphTokenUnavailable, Message: p.redactText(err.Error())}
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := p.client.Do(req)
	if err != nil {
		return MicrosoftGraphAccessToken{}, &MicrosoftGraphError{Evidence: MicrosoftGraphTokenUnavailable, Message: p.redactText(err.Error())}
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return MicrosoftGraphAccessToken{}, &MicrosoftGraphError{Evidence: MicrosoftGraphTokenUnavailable, Message: p.redactText(readErr.Error())}
	}
	if resp.StatusCode >= 400 {
		return MicrosoftGraphAccessToken{}, &MicrosoftGraphError{
			Evidence: MicrosoftGraphTokenUnavailable,
			Message:  p.redactText(fmt.Sprintf("Microsoft Graph token request failed with HTTP %d: %s", resp.StatusCode, microsoftGraphErrorDetail(body))),
		}
	}
	var payload struct {
		AccessToken string      `json:"access_token"`
		TokenType   string      `json:"token_type"`
		ExpiresIn   interface{} `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return MicrosoftGraphAccessToken{}, &MicrosoftGraphError{Evidence: MicrosoftGraphTokenUnavailable, Message: "Microsoft Graph token response was not valid JSON"}
	}
	payload.AccessToken = strings.TrimSpace(payload.AccessToken)
	if payload.AccessToken == "" {
		return MicrosoftGraphAccessToken{}, &MicrosoftGraphError{Evidence: MicrosoftGraphTokenUnavailable, Message: "Microsoft Graph token response did not include access_token"}
	}
	expiresIn, err := microsoftGraphExpiresIn(payload.ExpiresIn)
	if err != nil {
		return MicrosoftGraphAccessToken{}, &MicrosoftGraphError{Evidence: MicrosoftGraphTokenUnavailable, Message: "Microsoft Graph token response did not include a valid expires_in"}
	}
	tokenType := strings.TrimSpace(payload.TokenType)
	if tokenType == "" {
		tokenType = "Bearer"
	}
	return MicrosoftGraphAccessToken{
		AccessToken: payload.AccessToken,
		TokenType:   tokenType,
		ExpiresAt:   p.now().Add(time.Duration(expiresIn) * time.Second),
	}, nil
}

func normalizeMicrosoftGraphCredentials(creds MicrosoftGraphCredentials) MicrosoftGraphCredentials {
	creds.TenantID = strings.TrimSpace(creds.TenantID)
	creds.ClientID = strings.TrimSpace(creds.ClientID)
	creds.ClientSecret = strings.TrimSpace(creds.ClientSecret)
	creds.Scope = strings.TrimSpace(creds.Scope)
	if creds.Scope == "" {
		creds.Scope = DefaultMicrosoftGraphScope
	}
	creds.AuthorityURL = strings.TrimSpace(creds.AuthorityURL)
	if creds.AuthorityURL == "" {
		creds.AuthorityURL = DefaultMicrosoftGraphAuthorityURL
	}
	return creds
}

func microsoftGraphExpiresIn(value interface{}) (int64, error) {
	switch v := value.(type) {
	case float64:
		return int64(v), nil
	case string:
		var out int64
		_, err := fmt.Sscan(v, &out)
		return out, err
	default:
		return 0, errors.New("invalid expires_in")
	}
}

func microsoftGraphErrorDetail(body []byte) string {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		text := strings.TrimSpace(string(body))
		if text == "" {
			return "unknown error"
		}
		return text
	}
	if desc, ok := payload["error_description"].(string); ok && strings.TrimSpace(desc) != "" {
		return strings.TrimSpace(desc)
	}
	if nested, ok := payload["error"].(map[string]interface{}); ok {
		msg, _ := nested["message"].(string)
		code, _ := nested["code"].(string)
		switch {
		case strings.TrimSpace(msg) != "" && strings.TrimSpace(code) != "":
			return strings.TrimSpace(code) + ": " + strings.TrimSpace(msg)
		case strings.TrimSpace(msg) != "":
			return strings.TrimSpace(msg)
		}
	}
	if errText, ok := payload["error"].(string); ok && strings.TrimSpace(errText) != "" {
		return strings.TrimSpace(errText)
	}
	return "unknown error"
}

func (p *MicrosoftGraphTokenProvider) redactText(text string) string {
	return redactMicrosoftGraphText(text, p.redact...)
}

func redactMicrosoftGraphText(text string, secrets ...string) string {
	out := text
	for _, secret := range secrets {
		secret = strings.TrimSpace(secret)
		if secret != "" {
			out = strings.ReplaceAll(out, secret, "[redacted]")
		}
	}
	return microsoftGraphBearerPattern.ReplaceAllString(out, "Bearer [redacted]")
}
