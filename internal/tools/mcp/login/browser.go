package login

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/config"
)

// MCPBrowserLoginOptions makes the browser OAuth flow hermetic in tests. The
// default flow binds only to 127.0.0.1, but callers still inject BrowserOpen and
// HTTPClient so tests never open real browsers or contact live token endpoints.
type BrowserOptions struct {
	CallbackTimeout time.Duration
	BrowserOpen     func(context.Context, string) error
	HTTPClient      *http.Client
	Listen          func(context.Context) (net.Listener, error)
	State           string
	Now             func() time.Time
}

// BrowserMCPLoginFlow implements MCPLoginFlow with a localhost callback and an
// OAuth authorization-code token exchange. It is provider-neutral and stores no
// tokens itself; RunMCPLogin persists successful sessions.
type BrowserFlow struct {
	opts BrowserOptions
}

type callbackResult struct {
	code string
	err  error
}

func NewBrowserFlow(opts BrowserOptions) *BrowserFlow {
	return &BrowserFlow{opts: opts}
}

func (f *BrowserFlow) BuildAuthorizeURL(server config.MCPServerDefinition) (launchURL string, redirectURI string, err error) {
	return f.buildAuthorizeURL(server, "http://127.0.0.1:0/callback")
}

func (f *BrowserFlow) Login(ctx context.Context, server config.MCPServerDefinition) (*Session, error) {
	if f == nil {
		f = NewBrowserFlow(BrowserOptions{})
	}
	listen := f.opts.Listen
	if listen == nil {
		listen = func(context.Context) (net.Listener, error) { return net.Listen("tcp", "127.0.0.1:0") }
	}
	ln, err := listen(ctx)
	if err != nil {
		return nil, Result{Server: server.Name, Evidence: EvidencePortCollision, Message: sanitizeText(err.Error())}
	}
	defer ln.Close()

	redirectURI := "http://" + ln.Addr().String() + "/callback"
	launchURL, redirectURI, err := f.buildAuthorizeURL(server, redirectURI)
	if err != nil {
		return nil, Result{Server: server.Name, Evidence: EvidenceFlowFailed, Message: sanitizeText(err.Error())}
	}
	state := mustQueryValue(launchURL, "state")
	callbacks := make(chan callbackResult, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("state"); got != state {
			if !deliverCallbackResult(callbacks, callbackResult{err: Result{Server: server.Name, Evidence: EvidenceFlowFailed, Message: "OAuth state mismatch"}}) {
				w.WriteHeader(http.StatusConflict)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if got := strings.TrimSpace(q.Get("redirect_uri")); got != "" && got != redirectURI {
			if !deliverCallbackResult(callbacks, callbackResult{err: Result{Server: server.Name, Evidence: EvidenceRedirectURIMismatch, Message: "callback redirect_uri did not match launched redirect_uri"}}) {
				w.WriteHeader(http.StatusConflict)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		code := strings.TrimSpace(q.Get("code"))
		if code == "" {
			if !deliverCallbackResult(callbacks, callbackResult{err: Result{Server: server.Name, Evidence: EvidenceFlowFailed, Message: "OAuth callback missing code"}}) {
				w.WriteHeader(http.StatusConflict)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if !deliverCallbackResult(callbacks, callbackResult{code: code}) {
			w.WriteHeader(http.StatusConflict)
			return
		}
		_, _ = w.Write([]byte("Gormes MCP login received. You can close this window."))
	})
	srv := &http.Server{Handler: mux}
	serveDone := make(chan struct{}, 1)
	go func() {
		_ = srv.Serve(ln)
		close(serveDone)
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		<-serveDone
	}()

	open := f.opts.BrowserOpen
	if open == nil {
		return nil, Result{Server: server.Name, Evidence: EvidenceBrowserUnavailable, Message: "browser open hook is not configured"}
	}
	if err := open(ctx, launchURL); err != nil {
		return nil, Result{Server: server.Name, Evidence: EvidenceBrowserUnavailable, Message: sanitizeText(err.Error())}
	}

	timeout := f.opts.CallbackTimeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case result := <-callbacks:
		if result.err != nil {
			return nil, result.err
		}
		return f.exchangeCode(ctx, server, result.code, redirectURI)
	case <-timer.C:
		return nil, Result{Server: server.Name, Evidence: EvidenceCallbackTimeout, Message: "timed out waiting for OAuth callback"}
	case <-ctx.Done():
		return nil, Result{Server: server.Name, Evidence: EvidenceCallbackTimeout, Message: sanitizeText(ctx.Err().Error())}
	}
}

func deliverCallbackResult(callbacks chan<- callbackResult, result callbackResult) bool {
	select {
	case callbacks <- result:
		return true
	default:
		return false
	}
}

func (f *BrowserFlow) buildAuthorizeURL(server config.MCPServerDefinition, redirectURI string) (string, string, error) {
	base, err := url.Parse(strings.TrimRight(server.URL, "/"))
	if err != nil || base.Scheme == "" || base.Host == "" {
		return "", "", fmt.Errorf("invalid MCP OAuth URL")
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/authorize"
	q := base.Query()
	q.Set("response_type", "code")
	q.Set("client_id", FirstNonEmpty(server.Name, "gormes"))
	q.Set("redirect_uri", redirectURI)
	q.Set("state", f.state())
	base.RawQuery = q.Encode()
	return base.String(), redirectURI, nil
}

func (f *BrowserFlow) exchangeCode(ctx context.Context, server config.MCPServerDefinition, code, redirectURI string) (*Session, error) {
	endpoint, err := url.Parse(strings.TrimRight(server.URL, "/") + "/token")
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" {
		return nil, Result{Server: server.Name, Evidence: EvidenceFlowFailed, Message: "invalid token endpoint"}
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := f.opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, Result{Server: server.Name, Evidence: EvidenceTokenExchangeFailed, Message: sanitizeText(err.Error())}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, Result{Server: server.Name, Evidence: EvidenceTokenExchangeFailed, Message: "token exchange failed with status " + strconv.Itoa(resp.StatusCode)}
	}
	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, Result{Server: server.Name, Evidence: EvidenceTokenExchangeFailed, Message: "token exchange returned invalid JSON"}
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return nil, Result{Server: server.Name, Evidence: EvidenceTokenExchangeFailed, Message: "token exchange returned no access token"}
	}
	now := time.Now
	if f.opts.Now != nil {
		now = f.opts.Now
	}
	session := &Session{AccessToken: payload.AccessToken, RefreshToken: payload.RefreshToken, Scope: payload.Scope, Issuer: strings.TrimRight(server.URL, "/")}
	if payload.ExpiresIn > 0 {
		session.ExpiresAt = now().UTC().Add(time.Duration(payload.ExpiresIn) * time.Second)
	}
	return session, nil
}

func (f *BrowserFlow) state() string {
	if f != nil && strings.TrimSpace(f.opts.State) != "" {
		return strings.TrimSpace(f.opts.State)
	}
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err == nil {
		return "gormes-" + hex.EncodeToString(buf[:])
	}
	return "gormes-" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

func mustQueryValue(rawURL, key string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Query().Get(key)
}

var _ Flow = (*BrowserFlow)(nil)
