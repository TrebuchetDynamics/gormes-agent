package mcplogin

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/configwriter"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type testFlow struct {
	calls   int
	session *tools.MCPSession
	err     error
}

func (f *testFlow) Login(context.Context, tools.MCPServerDefinition) (*tools.MCPSession, error) {
	f.calls++
	return f.session, f.err
}

func TestRunSuccessPrintsSafeEvidenceAndSavesSession(t *testing.T) {
	store := tools.NewMCPOAuthStore()
	flow := &testFlow{session: &tools.MCPSession{AccessToken: "plain-access-token", RefreshToken: "plain-refresh-token"}}
	var stdout strings.Builder
	err := Run(context.Background(), Runtime{
		LoadConfig: func() (tools.MCPConfigResolution, error) { return testResolution(), nil },
		Store:      store,
		Flow:       flow,
	}, Options{ServerName: "oauth", Stdout: &stdout})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if flow.calls != 1 {
		t.Fatalf("flow calls = %d, want 1", flow.calls)
	}
	if !strings.Contains(stdout.String(), "mcp_login_saved") || strings.Contains(stdout.String(), "plain-access-token") || strings.Contains(stdout.String(), "plain-refresh-token") {
		t.Fatalf("stdout not safe or missing evidence: %q", stdout.String())
	}
	if _, ok := store.Get("oauth"); !ok {
		t.Fatalf("expected session saved")
	}
}

func TestRunUnknownServerTextDoesNotPrintDuplicateError(t *testing.T) {
	var stdout strings.Builder
	err := Run(context.Background(), Runtime{
		LoadConfig: func() (tools.MCPConfigResolution, error) { return testResolution(), nil },
		Store:      tools.NewMCPOAuthStore(),
	}, Options{ServerName: "missing", Stdout: &stdout})
	if err == nil || !strings.Contains(err.Error(), "mcp_server_unknown") {
		t.Fatalf("err = %v, want mcp_server_unknown", err)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty error-path stdout", stdout.String())
	}
}

func TestRunJSONUnknownServerEmitsAvailableArray(t *testing.T) {
	var stdout strings.Builder
	err := Run(context.Background(), Runtime{
		LoadConfig: func() (tools.MCPConfigResolution, error) { return testResolution(), nil },
		Store:      tools.NewMCPOAuthStore(),
	}, Options{ServerName: "missing", JSON: true, Stdout: &stdout, Build: BuildProvenance{Version: "test", GitCommit: "abc"}})
	if err == nil || !strings.Contains(err.Error(), "mcp_server_unknown") {
		t.Fatalf("err = %v, want mcp_server_unknown", err)
	}
	out := stdout.String()
	for _, want := range []string{"\"version\": \"test\"", "\"server\": \"missing\"", "\"evidence\": \"mcp_server_unknown\"", "\"oauth\"", "\"stdio\""} {
		if !strings.Contains(out, want) {
			t.Fatalf("json output missing %q: %s", want, out)
		}
	}
}

func TestRunLoadConfigErrorIsTypedForExit2Wrapper(t *testing.T) {
	err := Run(context.Background(), Runtime{
		LoadConfig: func() (tools.MCPConfigResolution, error) { return tools.MCPConfigResolution{}, errors.New("boom") },
	}, Options{ServerName: "oauth"})
	if err == nil || !strings.Contains(err.Error(), "mcp_config_unavailable: boom") {
		t.Fatalf("err = %v, want config error", err)
	}
	if got := ExitCodeForError(err); got != 2 {
		t.Fatalf("ExitCodeForError = %d, want 2", got)
	}
}

func TestLoadDefaultMCPConfigReadsActiveGormesProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("HERMES_HOME", filepath.Join(t.TempDir(), "must-not-read"))
	path := filepath.Join(home, "config.toml")
	if err := configwriter.WriteTOMLAtomic(path, map[string]any{
		"mcp_servers": map[string]any{
			"linear": map[string]any{
				"url":     "https://mcp.linear.app/mcp",
				"enabled": true,
				"auth":    "oauth",
			},
		},
	}); err != nil {
		t.Fatalf("WriteTOMLAtomic: %v", err)
	}

	resolved, err := LoadDefaultMCPConfig()
	if err != nil {
		t.Fatalf("LoadDefaultMCPConfig: %v", err)
	}
	if len(resolved.Servers) != 1 || resolved.Servers[0].Name != "linear" || resolved.Servers[0].URL != "https://mcp.linear.app/mcp" {
		t.Fatalf("servers = %#v", resolved.Servers)
	}
}

func testResolution() tools.MCPConfigResolution {
	return tools.MCPConfigResolution{Servers: []tools.MCPServerDefinition{
		{Name: "oauth", Enabled: true, Transport: tools.MCPTransportHTTP, URL: "https://mcp.example/oauth", Headers: map[string]string{}},
		{Name: "stdio", Enabled: true, Transport: tools.MCPTransportStdio, Command: "mcp-server"},
	}}
}
