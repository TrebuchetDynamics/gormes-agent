package mcp

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
)

const (
	defaultOAuthRedirectHost = "127.0.0.1"
	oauthCallbackPath        = "/oauth2callback"
	oauthRefreshSkew         = 60 * time.Second
	oauthHTTPTimeout         = 20 * time.Second
	oauthCallbackTimeout     = 5 * time.Minute
)

var (
	ErrOAuthLoginRequired = errors.New("mcp: OAuth login required")

	oauthNow         = time.Now
	oauthOpenBrowser = openOAuthBrowserURL
)

type OAuthLoginOptions struct {
	ServerName            string
	AuthorizationEndpoint string
	TokenEndpoint         string
	ClientID              string
	ClientSecret          string
	Scope                 string
	RedirectPort          int
	OpenBrowser           bool
	NotifyURL             func(string)
}

type OAuthLoginResult struct {
	Path string
}

type OAuthTokenStore struct {
	serverName string
}

type oauthCredentials struct {
	AccessToken  string `json:"access_token,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresAt    int64  `json:"expires_at,omitempty"`
}

type oauthClientInfo struct {
	ClientID              string `json:"client_id,omitempty"`
	ClientSecret          string `json:"client_secret,omitempty"`
	Scope                 string `json:"scope,omitempty"`
	AuthorizationEndpoint string `json:"authorization_endpoint,omitempty"`
	TokenEndpoint         string `json:"token_endpoint,omitempty"`
}

type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type oauthCallbackResult struct {
	Code string
	Err  error
}

func NewOAuthTokenStore(serverName string) OAuthTokenStore {
	return OAuthTokenStore{serverName: sanitizeOAuthServerName(serverName)}
}

func (s OAuthTokenStore) TokenPath() string {
	return filepath.Join(mcpTokenDir(), s.serverName+".json")
}

func (s OAuthTokenStore) ClientPath() string {
	return filepath.Join(mcpTokenDir(), s.serverName+".client.json")
}

func (s OAuthTokenStore) LoadCredentials() (oauthCredentials, error) {
	var creds oauthCredentials
	if err := readOAuthJSONFile(s.TokenPath(), &creds); err != nil {
		return oauthCredentials{}, err
	}
	return creds, nil
}

func (s OAuthTokenStore) SaveCredentials(creds oauthCredentials) error {
	return saveOAuthJSONFile(s.TokenPath(), creds)
}

func (s OAuthTokenStore) LoadClientInfo() (oauthClientInfo, error) {
	var client oauthClientInfo
	if err := readOAuthJSONFile(s.ClientPath(), &client); err != nil {
		return oauthClientInfo{}, err
	}
	return client, nil
}

func (s OAuthTokenStore) SaveClientInfo(client oauthClientInfo) error {
	return saveOAuthJSONFile(s.ClientPath(), client)
}

func LoginOAuth(ctx context.Context, opts OAuthLoginOptions) (OAuthLoginResult, error) {
	if strings.TrimSpace(opts.ServerName) == "" {
		return OAuthLoginResult{}, errors.New("mcp: OAuth server name is required")
	}
	if strings.TrimSpace(opts.AuthorizationEndpoint) == "" {
		return OAuthLoginResult{}, errors.New("mcp: OAuth authorization endpoint is required")
	}
	if strings.TrimSpace(opts.TokenEndpoint) == "" {
		return OAuthLoginResult{}, errors.New("mcp: OAuth token endpoint is required")
	}
	if strings.TrimSpace(opts.ClientID) == "" {
		return OAuthLoginResult{}, errors.New("mcp: OAuth client ID is required")
	}

	verifier, challenge, err := generateOAuthPKCE()
	if err != nil {
		return OAuthLoginResult{}, err
	}
	state, err := oauthRandomString(24)
	if err != nil {
		return OAuthLoginResult{}, err
	}

	redirectURI, waitForCode, shutdown, err := startOAuthCallbackServer(state, opts.RedirectPort)
	if err != nil {
		return OAuthLoginResult{}, err
	}
	defer shutdown()

	authURL := buildOAuthAuthURL(opts, redirectURI, challenge, state)
	if opts.NotifyURL != nil {
		opts.NotifyURL(authURL)
	}
	if opts.OpenBrowser {
		_ = oauthOpenBrowser(authURL)
	}

	waitCtx, cancel := context.WithTimeout(ctx, oauthCallbackTimeout)
	defer cancel()
	code, err := waitForCode(waitCtx)
	if err != nil {
		return OAuthLoginResult{}, err
	}

	tokenResp, err := exchangeOAuthCode(ctx, opts, code, verifier, redirectURI)
	if err != nil {
		return OAuthLoginResult{}, err
	}

	store := NewOAuthTokenStore(opts.ServerName)
	if err := store.SaveClientInfo(oauthClientInfo{
		ClientID:              strings.TrimSpace(opts.ClientID),
		ClientSecret:          strings.TrimSpace(opts.ClientSecret),
		Scope:                 strings.TrimSpace(opts.Scope),
		AuthorizationEndpoint: strings.TrimSpace(opts.AuthorizationEndpoint),
		TokenEndpoint:         strings.TrimSpace(opts.TokenEndpoint),
	}); err != nil {
		return OAuthLoginResult{}, err
	}

	creds := oauthCredentials{
		AccessToken:  strings.TrimSpace(tokenResp.AccessToken),
		TokenType:    firstNonEmpty(strings.TrimSpace(tokenResp.TokenType), "Bearer"),
		RefreshToken: strings.TrimSpace(tokenResp.RefreshToken),
		ExpiresAt:    oauthExpiryUnixMilli(tokenResp.ExpiresIn),
	}
	if err := store.SaveCredentials(creds); err != nil {
		return OAuthLoginResult{}, err
	}

	return OAuthLoginResult{Path: store.TokenPath()}, nil
}

func LoadOAuthAccessToken(ctx context.Context, serverName string) (string, error) {
	store := NewOAuthTokenStore(serverName)
	creds, err := store.LoadCredentials()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(creds.AccessToken) == "" && strings.TrimSpace(creds.RefreshToken) == "" {
		return "", ErrOAuthLoginRequired
	}
	if !creds.needsRefresh(oauthNow()) {
		return strings.TrimSpace(creds.AccessToken), nil
	}
	if strings.TrimSpace(creds.RefreshToken) == "" {
		return "", errors.New("mcp: cached OAuth token expired and no refresh token is available")
	}

	client, err := store.LoadClientInfo()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(client.TokenEndpoint) == "" {
		return "", errors.New("mcp: cached OAuth client info is missing token_endpoint")
	}
	if strings.TrimSpace(client.ClientID) == "" {
		return "", errors.New("mcp: cached OAuth client info is missing client_id")
	}

	tokenResp, err := refreshOAuthAccessToken(ctx, client, creds.RefreshToken)
	if err != nil {
		return "", err
	}
	creds.AccessToken = strings.TrimSpace(tokenResp.AccessToken)
	creds.TokenType = firstNonEmpty(strings.TrimSpace(tokenResp.TokenType), creds.TokenType, "Bearer")
	creds.RefreshToken = firstNonEmpty(strings.TrimSpace(tokenResp.RefreshToken), creds.RefreshToken)
	creds.ExpiresAt = oauthExpiryUnixMilli(tokenResp.ExpiresIn)
	if err := store.SaveCredentials(creds); err != nil {
		return "", err
	}
	return creds.AccessToken, nil
}

func (c oauthCredentials) needsRefresh(now time.Time) bool {
	if strings.TrimSpace(c.RefreshToken) == "" {
		return strings.TrimSpace(c.AccessToken) == ""
	}
	if strings.TrimSpace(c.AccessToken) == "" || c.ExpiresAt == 0 {
		return true
	}
	return !now.Add(oauthRefreshSkew).Before(time.UnixMilli(c.ExpiresAt))
}

func buildOAuthAuthURL(opts OAuthLoginOptions, redirectURI, challenge, state string) string {
	values := url.Values{}
	values.Set("client_id", strings.TrimSpace(opts.ClientID))
	values.Set("redirect_uri", redirectURI)
	values.Set("response_type", "code")
	values.Set("state", state)
	values.Set("code_challenge", challenge)
	values.Set("code_challenge_method", "S256")
	if scope := strings.TrimSpace(opts.Scope); scope != "" {
		values.Set("scope", scope)
	}
	return strings.TrimSpace(opts.AuthorizationEndpoint) + "?" + values.Encode()
}

func exchangeOAuthCode(ctx context.Context, opts OAuthLoginOptions, code, verifier, redirectURI string) (oauthTokenResponse, error) {
	values := url.Values{}
	values.Set("grant_type", "authorization_code")
	values.Set("code", strings.TrimSpace(code))
	values.Set("code_verifier", verifier)
	values.Set("client_id", strings.TrimSpace(opts.ClientID))
	values.Set("redirect_uri", redirectURI)
	if secret := strings.TrimSpace(opts.ClientSecret); secret != "" {
		values.Set("client_secret", secret)
	}
	return oauthTokenRequest(ctx, strings.TrimSpace(opts.TokenEndpoint), values)
}

func refreshOAuthAccessToken(ctx context.Context, client oauthClientInfo, refreshToken string) (oauthTokenResponse, error) {
	values := url.Values{}
	values.Set("grant_type", "refresh_token")
	values.Set("refresh_token", strings.TrimSpace(refreshToken))
	values.Set("client_id", strings.TrimSpace(client.ClientID))
	if secret := strings.TrimSpace(client.ClientSecret); secret != "" {
		values.Set("client_secret", secret)
	}
	return oauthTokenRequest(ctx, strings.TrimSpace(client.TokenEndpoint), values)
}

func oauthTokenRequest(ctx context.Context, endpoint string, values url.Values) (oauthTokenResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, oauthHTTPTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return oauthTokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return oauthTokenResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return oauthTokenResponse{}, err
	}
	if resp.StatusCode >= 300 {
		return oauthTokenResponse{}, fmt.Errorf("mcp: OAuth token endpoint returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tokenResp oauthTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return oauthTokenResponse{}, fmt.Errorf("decode OAuth token response: %w", err)
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return oauthTokenResponse{}, errors.New("mcp: OAuth token response missing access_token")
	}
	return tokenResp, nil
}

func startOAuthCallbackServer(state string, redirectPort int) (string, func(context.Context) (string, error), func(), error) {
	port := "0"
	if redirectPort > 0 {
		port = fmt.Sprintf("%d", redirectPort)
	}
	listener, err := net.Listen("tcp", net.JoinHostPort(defaultOAuthRedirectHost, port))
	if err != nil && redirectPort > 0 {
		listener, err = net.Listen("tcp", net.JoinHostPort(defaultOAuthRedirectHost, "0"))
	}
	if err != nil {
		return "", nil, nil, err
	}

	resultCh := make(chan oauthCallbackResult, 1)
	var once sync.Once
	sendResult := func(result oauthCallbackResult) {
		once.Do(func() {
			resultCh <- result
		})
	}

	mux := http.NewServeMux()
	mux.HandleFunc(oauthCallbackPath, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		switch {
		case query.Get("state") != state:
			http.Error(w, "MCP OAuth state mismatch", http.StatusBadRequest)
			sendResult(oauthCallbackResult{Err: errors.New("mcp: OAuth state mismatch")})
		case strings.TrimSpace(query.Get("error")) != "":
			http.Error(w, "MCP OAuth authorization failed", http.StatusBadRequest)
			sendResult(oauthCallbackResult{Err: fmt.Errorf("mcp: OAuth authorization failed: %s", query.Get("error"))})
		case strings.TrimSpace(query.Get("code")) == "":
			http.Error(w, "MCP OAuth callback missing code", http.StatusBadRequest)
			sendResult(oauthCallbackResult{Err: errors.New("mcp: OAuth callback missing code")})
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.WriteString(w, "<html><body><h1>Signed in.</h1><p>You can return to Gormes.</p></body></html>")
			sendResult(oauthCallbackResult{Code: query.Get("code")})
		}
	})

	server := &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(listener)
	}()

	addr := listener.Addr().(*net.TCPAddr)
	redirectURI := fmt.Sprintf("http://%s:%d%s", defaultOAuthRedirectHost, addr.Port, oauthCallbackPath)
	waitForCode := func(ctx context.Context) (string, error) {
		select {
		case result := <-resultCh:
			return result.Code, result.Err
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		_ = listener.Close()
	}
	return redirectURI, waitForCode, shutdown, nil
}

func generateOAuthPKCE() (string, string, error) {
	verifier, err := oauthRandomString(64)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func oauthRandomString(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func oauthExpiryUnixMilli(expiresInSeconds int64) int64 {
	if expiresInSeconds < 60 {
		expiresInSeconds = 60
	}
	return oauthNow().Add(time.Duration(expiresInSeconds) * time.Second).UnixMilli()
}

func mcpTokenDir() string {
	return filepath.Join(config.CrashLogDir(), "mcp-tokens")
}

func sanitizeOAuthServerName(serverName string) string {
	name := strings.TrimSpace(serverName)
	if name == "" {
		return "default"
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		out = "default"
	}
	if len(out) > 128 {
		out = out[:128]
	}
	return out
}

func readOAuthJSONFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func saveOAuthJSONFile(path string, body any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	payload, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	payload = append(payload, '\n')

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp.*")
	if err != nil {
		return fmt.Errorf("create temp %s: %w", path, err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()
	if _, err := tmp.Write(payload); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp %s: %w", path, err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return fmt.Errorf("chmod %s: %w", path, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename %s: %w", path, err)
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func openOAuthBrowserURL(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	return cmd.Start()
}
