package tools

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeMCPLoginFlow struct {
	calls   int
	session *MCPSession
	err     error
}

func (f *fakeMCPLoginFlow) Login(ctx context.Context, server MCPServerDefinition) (*MCPSession, error) {
	f.calls++
	return f.session, f.err
}

func TestMCPLoginInterfaceContract(t *testing.T) {
	var _ MCPLoginFlow = (*fakeMCPLoginFlow)(nil)
	var _ MCPLoginFlow = NoninteractiveLoginFlow()
}

func TestMCPLoginNoninteractiveDefaultReturnsTypedEvidence(t *testing.T) {
	store := NewMCPOAuthStore()
	res := MCPConfigResolution{Servers: []MCPServerDefinition{oauthServer("acme")}}
	result, err := RunMCPLogin(context.Background(), res, store, nil, "acme")
	if err != nil {
		t.Fatalf("RunMCPLogin returned error: %v", err)
	}
	if result.Evidence != MCPLoginEvidenceNoninteractiveRequired {
		t.Fatalf("evidence = %q, want %q", result.Evidence, MCPLoginEvidenceNoninteractiveRequired)
	}
	if !strings.Contains(result.Message, "gormes mcp remove") || !strings.Contains(result.Message, "browser-flow row") {
		t.Fatalf("guidance missing expected redirect text: %q", result.Message)
	}
	if _, ok := store.Get("acme"); ok {
		t.Fatalf("noninteractive default unexpectedly stored a session")
	}
}

func TestMCPLoginInjectedSuccessStoresSession(t *testing.T) {
	expires := time.Date(2027, 1, 2, 3, 4, 5, 0, time.UTC)
	flow := &fakeMCPLoginFlow{session: &MCPSession{
		AccessToken:  "plain-access-token",
		RefreshToken: "plain-refresh-token",
		Scope:        "tools.read",
		Issuer:       "https://issuer.example",
		ExpiresAt:    expires,
	}}
	store := NewMCPOAuthStore()
	res := MCPConfigResolution{Servers: []MCPServerDefinition{oauthServer("acme")}}
	result, err := RunMCPLogin(context.Background(), res, store, flow, "acme")
	if err != nil {
		t.Fatalf("RunMCPLogin returned error: %v", err)
	}
	if flow.calls != 1 {
		t.Fatalf("flow calls = %d, want 1", flow.calls)
	}
	if result.Evidence != MCPLoginEvidenceSaved {
		t.Fatalf("evidence = %q, want %q", result.Evidence, MCPLoginEvidenceSaved)
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

func TestMCPLoginInjectedFailureEmitsTypedError(t *testing.T) {
	store := NewMCPOAuthStore()
	_ = store.Set("existing", MCPOAuthToken{AccessToken: "plain-existing-token"})
	flow := &fakeMCPLoginFlow{err: errors.New("upstream failed with access_token=plain-secret")}
	res := MCPConfigResolution{Servers: []MCPServerDefinition{oauthServer("acme")}}
	result, err := RunMCPLogin(context.Background(), res, store, flow, "acme")
	if err != nil {
		t.Fatalf("RunMCPLogin returned error: %v", err)
	}
	if result.Evidence != MCPLoginEvidenceFlowFailed {
		t.Fatalf("evidence = %q, want %q", result.Evidence, MCPLoginEvidenceFlowFailed)
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

func TestMCPLoginUnknownServer(t *testing.T) {
	flow := &fakeMCPLoginFlow{}
	res := MCPConfigResolution{Servers: []MCPServerDefinition{oauthServer("alpha"), oauthServer("beta")}}
	result, err := RunMCPLogin(context.Background(), res, NewMCPOAuthStore(), flow, "does-not-exist")
	if err != nil {
		t.Fatalf("RunMCPLogin returned error: %v", err)
	}
	if result.Evidence != MCPLoginEvidenceServerUnknown {
		t.Fatalf("evidence = %q, want %q", result.Evidence, MCPLoginEvidenceServerUnknown)
	}
	if got := strings.Join(result.Available, ","); got != "alpha,beta" {
		t.Fatalf("available = %q, want alpha,beta", got)
	}
	if flow.calls != 0 {
		t.Fatalf("unknown server invoked flow %d times", flow.calls)
	}
}

func TestMCPLoginRejectsNonOAuth(t *testing.T) {
	cases := []MCPServerDefinition{
		{Name: "header", Enabled: true, Transport: MCPTransportHTTP, URL: "https://mcp.example", Headers: map[string]string{"Authorization": "Bearer plain-token"}},
		{Name: "stdio", Enabled: true, Transport: MCPTransportStdio, Command: "mcp-server"},
	}
	for _, server := range cases {
		t.Run(server.Name, func(t *testing.T) {
			flow := &fakeMCPLoginFlow{session: &MCPSession{AccessToken: "plain-access-token"}}
			res := MCPConfigResolution{Servers: []MCPServerDefinition{server}}
			result, err := RunMCPLogin(context.Background(), res, NewMCPOAuthStore(), flow, server.Name)
			if err != nil {
				t.Fatalf("RunMCPLogin returned error: %v", err)
			}
			if result.Evidence != MCPLoginEvidenceAuthNotOAuth {
				t.Fatalf("evidence = %q, want %q", result.Evidence, MCPLoginEvidenceAuthNotOAuth)
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

func oauthServer(name string) MCPServerDefinition {
	return MCPServerDefinition{Name: name, Enabled: true, Transport: MCPTransportHTTP, URL: "https://mcp.example/" + name, Headers: map[string]string{}}
}
