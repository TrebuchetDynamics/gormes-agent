package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type commandMCPLoginFlow struct {
	calls   int
	session *tools.MCPSession
	err     error
}

func (f *commandMCPLoginFlow) Login(ctx context.Context, server tools.MCPServerDefinition) (*tools.MCPSession, error) {
	f.calls++
	return f.session, f.err
}

func TestMCPLoginCommandInjectedSuccess(t *testing.T) {
	store := tools.NewMCPOAuthStore()
	flow := &commandMCPLoginFlow{session: &tools.MCPSession{AccessToken: "plain-access-token", RefreshToken: "plain-refresh-token"}}
	cmd := newMCPCommandWithRuntime(mcpLoginRuntime{
		loadConfig: func() (tools.MCPConfigResolution, error) { return commandMCPResolution(), nil },
		store:      store,
		flow:       flow,
	})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "login", "oauth")
	if err != nil {
		t.Fatalf("Execute: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if flow.calls != 1 {
		t.Fatalf("flow calls = %d, want 1", flow.calls)
	}
	if !strings.Contains(stdout, "mcp_login_saved") || strings.Contains(stdout+stderr, "plain-access-token") || strings.Contains(stdout+stderr, "plain-refresh-token") {
		t.Fatalf("stdout/stderr not safe or missing evidence:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if _, ok := store.Get("oauth"); !ok {
		t.Fatalf("expected session saved")
	}
}

func TestMCPLoginCommandUnknownServerExit2(t *testing.T) {
	cmd := newMCPCommandWithRuntime(mcpLoginRuntime{
		loadConfig: func() (tools.MCPConfigResolution, error) { return commandMCPResolution(), nil },
		store:      tools.NewMCPOAuthStore(),
	})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "login", "missing")
	if err == nil || exitCodeFromError(err) != 2 {
		t.Fatalf("err=%v exit=%d stdout=%s stderr=%s, want exit 2", err, exitCodeFromError(err), stdout, stderr)
	}
	for _, want := range []string{"mcp_server_unknown", "oauth", "stdio"} {
		if !strings.Contains(stdout+stderr, want) {
			t.Fatalf("output missing %q:\nstdout=%s\nstderr=%s", want, stdout, stderr)
		}
	}
}

func TestMCPLoginCommandRejectsNonOAuth(t *testing.T) {
	flow := &commandMCPLoginFlow{session: &tools.MCPSession{AccessToken: "plain-access-token"}}
	cmd := newMCPCommandWithRuntime(mcpLoginRuntime{
		loadConfig: func() (tools.MCPConfigResolution, error) { return commandMCPResolution(), nil },
		store:      tools.NewMCPOAuthStore(),
		flow:       flow,
	})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "login", "stdio")
	if err == nil || exitCodeFromError(err) != 2 {
		t.Fatalf("err=%v exit=%d stdout=%s stderr=%s, want exit 2", err, exitCodeFromError(err), stdout, stderr)
	}
	if flow.calls != 0 {
		t.Fatalf("non-OAuth invoked flow %d times", flow.calls)
	}
	for _, want := range []string{"mcp_auth_not_oauth", "gormes mcp remove", "gormes mcp add"} {
		if !strings.Contains(stdout+stderr, want) {
			t.Fatalf("output missing %q:\nstdout=%s\nstderr=%s", want, stdout, stderr)
		}
	}
}

func TestMCPLoginCommandInjectedFailure(t *testing.T) {
	flow := &commandMCPLoginFlow{err: errors.New("access_token=plain-secret failed")}
	cmd := newMCPCommandWithRuntime(mcpLoginRuntime{
		loadConfig: func() (tools.MCPConfigResolution, error) { return commandMCPResolution(), nil },
		store:      tools.NewMCPOAuthStore(),
		flow:       flow,
	})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "login", "oauth")
	if err == nil || exitCodeFromError(err) != 2 {
		t.Fatalf("err=%v exit=%d stdout=%s stderr=%s, want exit 2", err, exitCodeFromError(err), stdout, stderr)
	}
	if !strings.Contains(stdout+stderr, "mcp_login_flow_failed") || strings.Contains(stdout+stderr, "plain-secret") || strings.Contains(stdout+stderr, "access_token") {
		t.Fatalf("failure output unsafe or missing evidence:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
}

func TestMCPLoginRootCommandRegistered(t *testing.T) {
	cmd := newRootCommandWithRuntime(rootRuntime{})
	mcp, _, err := cmd.Find([]string{"mcp", "login", "server"})
	if err != nil {
		t.Fatalf("Find mcp login: %v", err)
	}
	if mcp == nil || mcp.Use != "login <name>" {
		t.Fatalf("found command = %#v, want mcp login", mcp)
	}
}

func commandMCPResolution() tools.MCPConfigResolution {
	return tools.MCPConfigResolution{Servers: []tools.MCPServerDefinition{
		{Name: "oauth", Enabled: true, Transport: tools.MCPTransportHTTP, URL: "https://mcp.example/oauth", Headers: map[string]string{}},
		{Name: "stdio", Enabled: true, Transport: tools.MCPTransportStdio, Command: "mcp-server"},
	}}
}
