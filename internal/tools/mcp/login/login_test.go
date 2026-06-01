package login

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/oauth"
)

type fakeFlow struct {
	calls   int
	session *Session
	err     error
}

func (f *fakeFlow) Login(ctx context.Context, server config.MCPServerDefinition) (*Session, error) {
	f.calls++
	return f.session, f.err
}

func TestLoginInterfaceContract(t *testing.T) {
	var _ Flow = (*fakeFlow)(nil)
	var _ Flow = NoninteractiveFlow()
}

func TestLoginNoninteractiveDefaultReturnsTypedEvidence(t *testing.T) {
	store := oauth.NewStore()
	res := config.MCPConfigResolution{Servers: []config.MCPServerDefinition{oauthTestServer("acme")}}
	result, err := Run(context.Background(), res, store, nil, "acme")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Evidence != EvidenceNoninteractiveRequired {
		t.Fatalf("evidence = %q, want %q", result.Evidence, EvidenceNoninteractiveRequired)
	}
	if !strings.Contains(result.Message, "gormes mcp remove") || !strings.Contains(result.Message, "browser-flow row") {
		t.Fatalf("guidance missing expected redirect text: %q", result.Message)
	}
	if _, ok := store.Get("acme"); ok {
		t.Fatalf("noninteractive default unexpectedly stored a session")
	}
}

func TestLoginInjectedSuccessStoresSession(t *testing.T) {
	expires := time.Date(2027, 1, 2, 3, 4, 5, 0, time.UTC)
	flow := &fakeFlow{session: &Session{
		AccessToken:  "plain-access-token",
		RefreshToken: "plain-refresh-token",
		Scope:        "tools.read",
		Issuer:       "https://issuer.example",
		ExpiresAt:    expires,
	}}
	store := oauth.NewStore()
	res := config.MCPConfigResolution{Servers: []config.MCPServerDefinition{oauthTestServer("acme")}}
	result, err := Run(context.Background(), res, store, flow, "acme")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if flow.calls != 1 {
		t.Fatalf("flow calls = %d, want 1", flow.calls)
	}
	if result.Evidence != EvidenceSaved {
		t.Fatalf("evidence = %q, want %q", result.Evidence, EvidenceSaved)
	}
	tok, ok := store.Get("acme")
	if !ok {
		t.Fatalf("expected saved token")
	}
	if tok.AccessToken != "plain-access-token" || tok.RefreshToken != "plain-refresh-token" || tok.Scope != "tools.read" || tok.ExpiresAt != expires {
		t.Fatalf("stored token mismatch: %#v", tok)
	}
	if strings.Contains(result.Error(), "plain-access-token") || strings.Contains(result.Error(), "plain-refresh-token") {
		t.Fatalf("safe result leaked token material: %q", result.Error())
	}
}

func TestLoginInjectedFailureEmitsTypedError(t *testing.T) {
	store := oauth.NewStore()
	_ = store.Set("existing", oauth.Token{AccessToken: "plain-existing-token"})
	flow := &fakeFlow{err: errors.New("upstream failed with access_token=plain-secret")}
	res := config.MCPConfigResolution{Servers: []config.MCPServerDefinition{oauthTestServer("acme")}}
	result, err := Run(context.Background(), res, store, flow, "acme")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Evidence != EvidenceFlowFailed {
		t.Fatalf("evidence = %q, want %q", result.Evidence, EvidenceFlowFailed)
	}
	if strings.Contains(result.Error(), "plain-secret") || strings.Contains(result.Error(), "access_token") {
		t.Fatalf("failure output leaked credential-shaped detail: %q", result.Error())
	}
	if _, ok := store.Get("acme"); ok {
		t.Fatalf("failed login mutated target store slot")
	}
	if _, ok := store.Get("existing"); !ok {
		t.Fatalf("failed login removed unrelated store slot")
	}
}

func TestLoginUnknownServer(t *testing.T) {
	flow := &fakeFlow{}
	res := config.MCPConfigResolution{Servers: []config.MCPServerDefinition{oauthTestServer("alpha"), oauthTestServer("beta")}}
	result, err := Run(context.Background(), res, oauth.NewStore(), flow, "does-not-exist")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Evidence != EvidenceServerUnknown {
		t.Fatalf("evidence = %q, want %q", result.Evidence, EvidenceServerUnknown)
	}
	if got := strings.Join(result.Available, ","); got != "alpha,beta" {
		t.Fatalf("available = %q, want alpha,beta", got)
	}
	if flow.calls != 0 {
		t.Fatalf("unknown server invoked flow %d times", flow.calls)
	}
}

func TestLoginRejectsNonOAuth(t *testing.T) {
	cases := []config.MCPServerDefinition{
		{Name: "header", Enabled: true, Transport: config.MCPTransportHTTP, URL: "https://mcp.example", Headers: map[string]string{"Authorization": "Bearer plain-token"}},
		{Name: "stdio", Enabled: true, Transport: config.MCPTransportStdio, Command: "mcp-server"},
	}
	for _, server := range cases {
		t.Run(server.Name, func(t *testing.T) {
			flow := &fakeFlow{session: &Session{AccessToken: "plain-access-token"}}
			res := config.MCPConfigResolution{Servers: []config.MCPServerDefinition{server}}
			result, err := Run(context.Background(), res, oauth.NewStore(), flow, server.Name)
			if err != nil {
				t.Fatalf("Run returned error: %v", err)
			}
			if result.Evidence != EvidenceAuthNotOAuth {
				t.Fatalf("evidence = %q, want %q", result.Evidence, EvidenceAuthNotOAuth)
			}
			if !strings.Contains(result.Message, "gormes mcp remove") || !strings.Contains(result.Message, "gormes mcp add") {
				t.Fatalf("redirect missing remove/add guidance: %q", result.Message)
			}
			if flow.calls != 0 {
				t.Fatalf("non-OAuth server invoked flow %d times", flow.calls)
			}
		})
	}
}

func oauthTestServer(name string) config.MCPServerDefinition {
	return config.MCPServerDefinition{Name: name, Enabled: true, Transport: config.MCPTransportHTTP, URL: "https://mcp.example/" + name, Headers: map[string]string{}}
}
