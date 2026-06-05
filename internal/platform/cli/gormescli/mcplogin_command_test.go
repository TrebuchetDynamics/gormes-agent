package gormescli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

const testMCPVersion = "test-version"

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
	cmd := NewMCPCommandWithRuntime(MCPLoginRuntime{
		LoadConfig: func() (tools.MCPConfigResolution, error) { return commandMCPResolution(), nil },
		Store:      store,
		Flow:       flow,
	}, testMCPCommandOptions())
	stdout, stderr, err := executeMCPCommandForTest(cmd, "login", "oauth")
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
	cmd := NewMCPCommandWithRuntime(MCPLoginRuntime{
		LoadConfig: func() (tools.MCPConfigResolution, error) { return commandMCPResolution(), nil },
		Store:      tools.NewMCPOAuthStore(),
	}, testMCPCommandOptions())
	stdout, stderr, err := executeMCPCommandForTest(cmd, "login", "missing")
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
	cmd := NewMCPCommandWithRuntime(MCPLoginRuntime{
		LoadConfig: func() (tools.MCPConfigResolution, error) { return commandMCPResolution(), nil },
		Store:      tools.NewMCPOAuthStore(),
		Flow:       flow,
	}, testMCPCommandOptions())
	stdout, stderr, err := executeMCPCommandForTest(cmd, "login", "stdio")
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
	cmd := NewMCPCommandWithRuntime(MCPLoginRuntime{
		LoadConfig: func() (tools.MCPConfigResolution, error) { return commandMCPResolution(), nil },
		Store:      tools.NewMCPOAuthStore(),
		Flow:       flow,
	}, testMCPCommandOptions())
	stdout, stderr, err := executeMCPCommandForTest(cmd, "login", "oauth")
	if err == nil || exitCodeFromError(err) != 2 {
		t.Fatalf("err=%v exit=%d stdout=%s stderr=%s, want exit 2", err, exitCodeFromError(err), stdout, stderr)
	}
	if !strings.Contains(stdout+stderr, "mcp_login_flow_failed") || strings.Contains(stdout+stderr, "plain-secret") || strings.Contains(stdout+stderr, "access_token") {
		t.Fatalf("failure output unsafe or missing evidence:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
}

func TestMCPLoginDefaultIgnoresHermesConfig(t *testing.T) {
	root := t.TempDir()
	hermesHome := filepath.Join(root, "hermes")
	t.Setenv("HERMES_HOME", hermesHome)
	t.Setenv("HOME", filepath.Join(root, "home"))
	if err := os.MkdirAll(hermesHome, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hermesHome, "config.yaml"), []byte(`
mcp_servers:
  oauth:
    transport: http
    url: https://mcp.example/oauth
`), 0o600); err != nil {
		t.Fatal(err)
	}

	flow := &commandMCPLoginFlow{session: &tools.MCPSession{AccessToken: "plain-access-token"}}
	cmd := NewMCPCommandWithRuntime(MCPLoginRuntime{
		Store: tools.NewMCPOAuthStore(),
		Flow:  flow,
	}, testMCPCommandOptions())
	stdout, stderr, err := executeMCPCommandForTest(cmd, "login", "oauth")
	if err == nil || exitCodeFromError(err) != 2 {
		t.Fatalf("err=%v exit=%d stdout=%s stderr=%s, want exit 2", err, exitCodeFromError(err), stdout, stderr)
	}
	if flow.calls != 0 {
		t.Fatalf("default MCP login read Hermes config and invoked flow %d times", flow.calls)
	}
	if !strings.Contains(stdout+stderr, "mcp_server_unknown") {
		t.Fatalf("output missing mcp_server_unknown:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if strings.Contains(stdout+stderr, "noninteractive_required") || strings.Contains(stdout+stderr, "mcp_login_saved") {
		t.Fatalf("default MCP login used Hermes server config:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
}

// TestMCPLoginCommand_JSONEmitsStructuredOutcome proves
// `gormes mcp login <name> --json` returns
// `{build, server, evidence, message?, available?}` so fleet automation
// orchestrating MCP token refresh across machines can reason about
// every typed evidence value (saved/server_unknown/auth_not_oauth/...)
// without scraping `evidence=...` prose. Raw access/refresh tokens
// MUST never appear in stdout.
func TestMCPLoginCommand_JSONEmitsStructuredOutcome(t *testing.T) {
	store := tools.NewMCPOAuthStore()
	flow := &commandMCPLoginFlow{session: &tools.MCPSession{AccessToken: "plain-access-token", RefreshToken: "plain-refresh-token"}}
	cmd := NewMCPCommandWithRuntime(MCPLoginRuntime{
		LoadConfig: func() (tools.MCPConfigResolution, error) { return commandMCPResolution(), nil },
		Store:      store,
		Flow:       flow,
	}, testMCPCommandOptions())
	stdout, stderr, err := executeMCPCommandForTest(cmd, "login", "oauth", "--json")
	if err != nil {
		t.Fatalf("Execute mcp login --json: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.Contains(stdout+stderr, "plain-access-token") || strings.Contains(stdout+stderr, "plain-refresh-token") {
		t.Fatalf("mcp login --json LEAKED token:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Server   string `json:"server"`
		Evidence string `json:"evidence"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("mcp login --json must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Build.Version != testMCPVersion {
		t.Errorf("got.build.version = %q, want %q", got.Build.Version, testMCPVersion)
	}
	if got.Server != "oauth" {
		t.Errorf("server = %q, want oauth", got.Server)
	}
	if got.Evidence != "mcp_login_saved" {
		t.Errorf("evidence = %q, want %q", got.Evidence, "mcp_login_saved")
	}
}

// TestMCPLoginCommand_JSONUnknownServerEmitsAvailableArray proves
// the unknown-server path returns an `available` array so fleet
// automation hitting "did you mean?" cases can programmatically
// select an alternate server name.
func TestMCPLoginCommand_JSONUnknownServerEmitsAvailableArray(t *testing.T) {
	cmd := NewMCPCommandWithRuntime(MCPLoginRuntime{
		LoadConfig: func() (tools.MCPConfigResolution, error) { return commandMCPResolution(), nil },
		Store:      tools.NewMCPOAuthStore(),
	}, testMCPCommandOptions())
	stdout, _, err := executeMCPCommandForTest(cmd, "login", "missing", "--json")
	if err == nil || exitCodeFromError(err) != 2 {
		t.Fatalf("err=%v exit=%d stdout=%s, want exit 2", err, exitCodeFromError(err), stdout)
	}
	var got struct {
		Server    string   `json:"server"`
		Evidence  string   `json:"evidence"`
		Available []string `json:"available"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("mcp login --json (unknown server) must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if got.Evidence != "mcp_server_unknown" {
		t.Errorf("evidence = %q, want %q", got.Evidence, "mcp_server_unknown")
	}
	wantAvailable := map[string]bool{"oauth": false, "stdio": false}
	for _, name := range got.Available {
		if _, ok := wantAvailable[name]; ok {
			wantAvailable[name] = true
		}
	}
	for name, saw := range wantAvailable {
		if !saw {
			t.Errorf("available array missing %q; got %+v", name, got.Available)
		}
	}
}

// TestMCPLoginCommand_TextModeDoesNotDuplicateError pins the
// regression where `gormes mcp login <missing-server>` emitted the
// same error twice — once on stdout via fmt.Fprintln(result.Error())
// AND once on stderr as cobra's standard `Error: ...` rendering of
// the returned error. The contract: the evidence string must appear
// EXACTLY ONCE across stdout+stderr.
func TestMCPLoginCommand_TextModeDoesNotDuplicateError(t *testing.T) {
	cmd := NewMCPCommandWithRuntime(MCPLoginRuntime{
		LoadConfig: func() (tools.MCPConfigResolution, error) {
			return commandMCPResolution(), nil
		},
		Store: tools.NewMCPOAuthStore(),
	}, testMCPCommandOptions())
	stdout, stderr, err := executeMCPCommandForTest(cmd, "login", "missing")
	if err == nil {
		t.Fatalf("missing server must error; stdout=%s stderr=%s", stdout, stderr)
	}
	combined := stdout + "\n" + stderr
	count := strings.Count(combined, "mcp_server_unknown")
	if count != 1 {
		t.Fatalf("`mcp_server_unknown` must appear EXACTLY once across stdout+stderr (was %d times):\nstdout=%s\nstderr=%s", count, stdout, stderr)
	}
}

func TestMCPLoginRootCommandRegistered(t *testing.T) {
	factories := stubRootFactories()
	factories["mcp"] = func() *cobra.Command { return NewMCPCommand(testMCPCommandOptions()) }
	cmd := NewRootCommand(RootOptions{}, factories)
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

func testMCPCommandOptions() MCPCommandOptions {
	return MCPCommandOptions{
		BuildProvenance: func() BuildProvenance {
			return BuildProvenance{Version: testMCPVersion, GitCommit: "test-sha"}
		},
		ExitCodeError: NewExitCodeError,
	}
}

func executeMCPCommandForTest(cmd *cobra.Command, args ...string) (string, string, error) {
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}
