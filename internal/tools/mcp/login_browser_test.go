package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestMCPLoginBrowserHookRecordsLaunchURL(t *testing.T) {
	flow := NewBrowserMCPLoginFlow(MCPBrowserLoginOptions{
		BrowserOpen: func(ctx context.Context, launchURL string) error { return nil },
		HTTPClient:  httptestTokenClient(t, func(r *http.Request) {}),
	})
	server := oauthServer("acme")
	server.URL = "https://mcp.example/oauth"

	launchURL, redirectURI, err := flow.BuildAuthorizeURL(server)
	if err != nil {
		t.Fatalf("BuildAuthorizeURL() error = %v", err)
	}
	if !strings.HasPrefix(redirectURI, "http://127.0.0.1:") {
		t.Fatalf("redirectURI = %q, want localhost callback", redirectURI)
	}
	parsed, err := url.Parse(launchURL)
	if err != nil {
		t.Fatalf("launch URL invalid: %v", err)
	}
	if parsed.Path != "/oauth/authorize" {
		t.Fatalf("authorize path = %q, want /oauth/authorize", parsed.Path)
	}
	if got := parsed.Query().Get("redirect_uri"); got != redirectURI {
		t.Fatalf("redirect_uri = %q, want %q", got, redirectURI)
	}
	if parsed.Query().Get("state") == "" {
		t.Fatalf("state missing from launch URL: %s", launchURL)
	}
}

func TestMCPLoginBrowserTokenExchangeStoresSession(t *testing.T) {
	var exchangedCode string
	issuer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mcp/token":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("ParseForm: %v", err)
			}
			exchangedCode = r.Form.Get("code")
			if r.Form.Get("grant_type") != "authorization_code" {
				t.Fatalf("grant_type = %q", r.Form.Get("grant_type"))
			}
			if !strings.HasPrefix(r.Form.Get("redirect_uri"), "http://127.0.0.1:") {
				t.Fatalf("redirect_uri = %q, want localhost", r.Form.Get("redirect_uri"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token":  "plain-access-token",
				"refresh_token": "plain-refresh-token",
				"scope":         "tools.read tools.write",
				"expires_in":    3600,
			})
		default:
			t.Fatalf("unexpected request path %s", r.URL.Path)
		}
	}))
	defer issuer.Close()

	var launched string
	flow := NewBrowserMCPLoginFlow(MCPBrowserLoginOptions{
		CallbackTimeout: 2 * time.Second,
		BrowserOpen: func(ctx context.Context, launchURL string) error {
			launched = launchURL
			parsed, err := url.Parse(launchURL)
			if err != nil {
				return err
			}
			callbackURL := parsed.Query().Get("redirect_uri") + "?code=plain-code&state=" + url.QueryEscape(parsed.Query().Get("state"))
			go func() { _, _ = http.Get(callbackURL) }()
			return nil
		},
		HTTPClient: issuer.Client(),
	})
	server := oauthServer("acme")
	server.URL = issuer.URL + "/mcp"
	store := NewMCPOAuthStore()
	result, err := RunMCPLogin(context.Background(), MCPConfigResolution{Servers: []MCPServerDefinition{server}}, store, flow, "acme")
	if err != nil {
		t.Fatalf("RunMCPLogin returned error: %v", err)
	}
	if result.Evidence != MCPLoginEvidenceSaved {
		t.Fatalf("evidence = %q, want %q message=%q", result.Evidence, MCPLoginEvidenceSaved, result.Message)
	}
	if launched == "" {
		t.Fatal("browser open hook was not called")
	}
	if exchangedCode != "plain-code" {
		t.Fatalf("exchanged code = %q, want plain-code", exchangedCode)
	}
	tok, ok := store.Get("acme")
	if !ok {
		t.Fatal("OAuth token was not stored")
	}
	if tok.AccessToken != "plain-access-token" || tok.RefreshToken != "plain-refresh-token" || tok.Scope != "tools.read tools.write" {
		t.Fatalf("stored token mismatch: %#v", tok)
	}
	if strings.Contains(result.Error(), "plain-access-token") || strings.Contains(result.Error(), "plain-refresh-token") {
		t.Fatalf("result leaked token material: %q", result.Error())
	}
}

func TestMCPLoginBrowserRedirectURIMismatchTypedEvidence(t *testing.T) {
	store := NewMCPOAuthStore()
	_ = store.Set("acme", MCPOAuthToken{AccessToken: "plain-existing-token"})
	before, _ := store.Get("acme")
	flow := NewBrowserMCPLoginFlow(MCPBrowserLoginOptions{
		CallbackTimeout: 2 * time.Second,
		BrowserOpen: func(ctx context.Context, launchURL string) error {
			parsed, err := url.Parse(launchURL)
			if err != nil {
				return err
			}
			callbackURL := parsed.Query().Get("redirect_uri") + "?code=plain-code&state=" + url.QueryEscape(parsed.Query().Get("state")) + "&redirect_uri=http%3A%2F%2F127.0.0.1%3A1%2Fcallback"
			go func() { _, _ = http.Get(callbackURL) }()
			return nil
		},
	})
	server := oauthServer("acme")
	result, err := RunMCPLogin(context.Background(), MCPConfigResolution{Servers: []MCPServerDefinition{server}}, store, flow, "acme")
	if err != nil {
		t.Fatalf("RunMCPLogin returned error: %v", err)
	}
	if result.Evidence != MCPLoginEvidence("mcp_login_redirect_uri_mismatch") {
		t.Fatalf("evidence = %q, want redirect mismatch", result.Evidence)
	}
	after, _ := store.Get("acme")
	if after != before {
		t.Fatalf("failed login mutated existing token: before=%#v after=%#v", before, after)
	}
}

func TestMCPLoginBrowserTokenExchangeFailureTypedEvidence(t *testing.T) {
	issuer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "provider body with access_token=plain-secret", http.StatusBadGateway)
	}))
	defer issuer.Close()

	flow := NewBrowserMCPLoginFlow(MCPBrowserLoginOptions{
		CallbackTimeout: 2 * time.Second,
		BrowserOpen: func(ctx context.Context, launchURL string) error {
			parsed, err := url.Parse(launchURL)
			if err != nil {
				return err
			}
			callbackURL := parsed.Query().Get("redirect_uri") + "?code=plain-code&state=" + url.QueryEscape(parsed.Query().Get("state"))
			go func() { _, _ = http.Get(callbackURL) }()
			return nil
		},
		HTTPClient: issuer.Client(),
	})
	server := oauthServer("acme")
	server.URL = issuer.URL
	result, err := RunMCPLogin(context.Background(), MCPConfigResolution{Servers: []MCPServerDefinition{server}}, NewMCPOAuthStore(), flow, "acme")
	if err != nil {
		t.Fatalf("RunMCPLogin returned error: %v", err)
	}
	if result.Evidence != MCPLoginEvidence("mcp_login_token_exchange_failed") {
		t.Fatalf("evidence = %q, want token exchange failure", result.Evidence)
	}
	if strings.Contains(result.Error(), "plain-secret") || strings.Contains(result.Error(), "access_token") {
		t.Fatalf("token exchange failure leaked provider body: %q", result.Error())
	}
}

func TestMCPLoginBrowserDuplicateCallbackDoesNotBlockExchange(t *testing.T) {
	issuer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "plain-access-token",
				"expires_in":   3600,
			})
		default:
			t.Fatalf("unexpected request path %s", r.URL.Path)
		}
	}))
	defer issuer.Close()

	flow := NewBrowserMCPLoginFlow(MCPBrowserLoginOptions{
		CallbackTimeout: 2 * time.Second,
		BrowserOpen: func(ctx context.Context, launchURL string) error {
			parsed, err := url.Parse(launchURL)
			if err != nil {
				return err
			}
			callbackURL := parsed.Query().Get("redirect_uri") + "?code=plain-code&state=" + url.QueryEscape(parsed.Query().Get("state"))
			client := &http.Client{Timeout: 200 * time.Millisecond}
			for i := 0; i < 2; i++ {
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, callbackURL, nil)
				if err != nil {
					return err
				}
				resp, err := client.Do(req)
				if err != nil {
					return err
				}
				_ = resp.Body.Close()
			}
			return nil
		},
		HTTPClient: issuer.Client(),
	})
	server := oauthServer("acme")
	server.URL = issuer.URL
	result, err := RunMCPLogin(context.Background(), MCPConfigResolution{Servers: []MCPServerDefinition{server}}, NewMCPOAuthStore(), flow, "acme")
	if err != nil {
		t.Fatalf("RunMCPLogin returned error: %v", err)
	}
	if result.Evidence != MCPLoginEvidenceSaved {
		t.Fatalf("evidence = %q, want saved; message=%q", result.Evidence, result.Message)
	}
}

func TestMCPLoginBrowserPortCollisionTypedEvidence(t *testing.T) {
	flow := NewBrowserMCPLoginFlow(MCPBrowserLoginOptions{
		Listen: func(ctx context.Context) (net.Listener, error) {
			return nil, errors.New("bind: address already in use")
		},
	})
	server := oauthServer("acme")
	result, err := RunMCPLogin(context.Background(), MCPConfigResolution{Servers: []MCPServerDefinition{server}}, NewMCPOAuthStore(), flow, "acme")
	if err != nil {
		t.Fatalf("RunMCPLogin returned error: %v", err)
	}
	if result.Evidence != MCPLoginEvidence("mcp_login_port_collision") {
		t.Fatalf("evidence = %q, want port collision", result.Evidence)
	}
}

func TestMCPLoginBrowserCallbackTimeoutTypedEvidence(t *testing.T) {
	flow := NewBrowserMCPLoginFlow(MCPBrowserLoginOptions{
		CallbackTimeout: 20 * time.Millisecond,
		BrowserOpen:     func(context.Context, string) error { return nil },
	})
	server := oauthServer("acme")
	result, err := RunMCPLogin(context.Background(), MCPConfigResolution{Servers: []MCPServerDefinition{server}}, NewMCPOAuthStore(), flow, "acme")
	if err != nil {
		t.Fatalf("RunMCPLogin returned error: %v", err)
	}
	if result.Evidence != MCPLoginEvidence("mcp_login_callback_timeout") {
		t.Fatalf("evidence = %q, want timeout", result.Evidence)
	}
}

func httptestTokenClient(t *testing.T, check func(*http.Request)) *http.Client {
	t.Helper()
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		check(r)
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Header: make(http.Header), Request: r}, nil
	})}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
