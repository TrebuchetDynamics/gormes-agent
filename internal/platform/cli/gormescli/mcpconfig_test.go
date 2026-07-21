package gormescli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/configwriter"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/callresult"
	mcpconfig "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/descriptor"
	mcpprobe "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/probe"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/remote"
	mcpruntime "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/runtime"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit"
)

type testMCPProbeSession struct {
	tools    []descriptor.RawTool
	listErr  error
	closeErr error
	closed   *bool
}

func (session *testMCPProbeSession) ListTools(context.Context) ([]descriptor.RawTool, error) {
	return session.tools, session.listErr
}

func (session *testMCPProbeSession) Close() error {
	if session.closed != nil {
		*session.closed = true
	}
	return session.closeErr
}

type testMCPConfigureRuntimeSession struct {
	tools []descriptor.RawTool
}

func (session *testMCPConfigureRuntimeSession) ListTools(context.Context) ([]descriptor.RawTool, error) {
	return append([]descriptor.RawTool(nil), session.tools...), nil
}

func (*testMCPConfigureRuntimeSession) CallTool(context.Context, string, map[string]any) (callresult.Result, error) {
	return callresult.Result{}, nil
}

func (*testMCPConfigureRuntimeSession) Close() error { return nil }

func TestMCPTestHTTPUsesInjectedConnectorAndSanitizesTools(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "profile", "config.toml")
	if err := configwriter.WriteTOMLAtomic(configPath, map[string]any{
		"mcp_servers": map[string]any{
			"ink": map[string]any{
				"url":             "https://private.example/mcp?token=must-not-leak",
				"headers":         map[string]any{"Authorization": "Bearer header-secret-value"},
				"connect_timeout": "2s",
				"enabled":         true,
			},
		},
	}); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	closed := false
	called := false
	opts := testMCPCommandOptions()
	opts.MCPConfigPath = func() string { return configPath }
	opts.MCPProbeConnector = mcpprobe.Connector(func(_ context.Context, def mcpconfig.MCPServerDefinition) (mcpprobe.Session, error) {
		called = true
		if def.Name != "ink" || def.Transport != mcpconfig.MCPTransportHTTP || def.Headers["Authorization"] != "Bearer header-secret-value" {
			t.Fatalf("connector definition = %+v", def)
		}
		return &testMCPProbeSession{closed: &closed, tools: []descriptor.RawTool{
			{Name: "zeta", Description: "API_KEY=super-secret-value"},
			{Name: "\x1b]8;;https://evil.test\x07alpha\x1b]8;;\x07", Description: "safe\nline"},
		}}, nil
	})
	stdout, stderr, err := executeMCPCommandForTest(NewMCPCommand(opts), "test", "ink", "--timeout", "1s", "--json")
	if err != nil {
		t.Fatalf("mcp test: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !called || !closed {
		t.Fatalf("connector called=%v session closed=%v", called, closed)
	}
	var report struct {
		Build      BuildProvenance `json:"build"`
		Action     string          `json:"action"`
		Name       string          `json:"name"`
		Evidence   string          `json:"evidence"`
		Transport  string          `json:"transport"`
		ToolsCount int             `json:"tools_count"`
		Truncated  bool            `json:"truncated"`
		Tools      []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"tools"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &report); jsonErr != nil {
		t.Fatalf("JSON invalid: %v\n%s", jsonErr, stdout)
	}
	if report.Build.Version != testMCPVersion || report.Action != "mcp_test" || report.Name != "ink" || report.Evidence != "connected" || report.Transport != "http" || report.ToolsCount != 2 || report.Truncated || len(report.Tools) != 2 {
		t.Fatalf("report = %+v", report)
	}
	if report.Tools[0].Name != "alpha" || report.Tools[0].Description != "safe line" || report.Tools[1].Name != "zeta" || !strings.Contains(report.Tools[1].Description, "[redacted]") {
		t.Fatalf("tools = %+v", report.Tools)
	}
	combined := stdout + stderr
	for _, forbidden := range []string{"private.example", "must-not-leak", "header-secret-value", "super-secret-value", "https://evil.test", "\x1b", "\x07", configPath} {
		if strings.Contains(combined, forbidden) {
			t.Fatalf("mcp test output leaked %q: %s", forbidden, combined)
		}
	}
}

func TestMCPTestDefaultConnectorReachesHermeticOfficialSDKServer(t *testing.T) {
	sdkServer := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "fixture", Version: "1"}, nil)
	sdkServer.AddTool(&mcpsdk.Tool{
		Name:        "deploy",
		Description: "API_KEY=server-secret-value",
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}, func(context.Context, *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		return &mcpsdk.CallToolResult{}, nil
	})
	handler := mcpsdk.NewStreamableHTTPHandler(func(request *http.Request) *mcpsdk.Server {
		if got := request.Header.Get("Authorization"); got != "Bearer config-secret-value" {
			t.Errorf("Authorization = %q", got)
		}
		return sdkServer
	}, &mcpsdk.StreamableHTTPOptions{JSONResponse: true})
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := configwriter.WriteTOMLAtomic(configPath, map[string]any{
		"mcp_servers": map[string]any{"ink": map[string]any{
			"url":     httpServer.URL + "/private?token=config-url-secret",
			"headers": map[string]any{"Authorization": "Bearer config-secret-value"},
			"enabled": true,
		}},
	}); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	before, _ := os.ReadFile(configPath)
	opts := testMCPCommandOptions()
	opts.MCPConfigPath = func() string { return configPath }
	stdout, stderr, err := executeMCPCommandForTest(NewMCPCommand(opts), "test", "ink", "--timeout", "2s", "--json")
	if err != nil {
		t.Fatalf("mcp test default connector: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var report mcpTestReportJSON
	if jsonErr := json.Unmarshal([]byte(stdout), &report); jsonErr != nil || report.Evidence != "connected" || report.ToolsCount != 1 || len(report.Tools) != 1 || report.Tools[0].Name != "deploy" {
		t.Fatalf("report=%+v jsonErr=%v stdout=%s", report, jsonErr, stdout)
	}
	combined := stdout + stderr
	for _, forbidden := range []string{"config-secret-value", "config-url-secret", "server-secret-value", "/private", configPath} {
		if strings.Contains(combined, forbidden) {
			t.Fatalf("default connector output leaked %q: %s", forbidden, combined)
		}
	}
	after, _ := os.ReadFile(configPath)
	if string(after) != string(before) {
		t.Fatalf("probe changed config:\n%s", after)
	}
}

func TestMCPTestPreNetworkGatesAreNonMutating(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := configwriter.WriteTOMLAtomic(configPath, map[string]any{
		"mcp_servers": map[string]any{
			"disabled": map[string]any{"url": "https://disabled.example/private", "enabled": false},
			"oauth":    map[string]any{"url": "https://oauth.example/private", "auth": "oauth", "enabled": true},
			"stdio":    map[string]any{"command": "/private/bin/server", "args": []string{"secret-arg"}, "enabled": true},
		},
	}); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	before, _ := os.ReadFile(configPath)
	calls := 0
	opts := testMCPCommandOptions()
	opts.MCPConfigPath = func() string { return configPath }
	opts.MCPProbeConnector = func(context.Context, mcpconfig.MCPServerDefinition) (mcpprobe.Session, error) {
		calls++
		return nil, errors.New("must not connect")
	}
	for _, test := range []struct {
		name     string
		evidence string
	}{
		{name: "missing", evidence: "not_found"},
		{name: "bad/name", evidence: "invalid_input"},
		{name: "disabled", evidence: "disabled"},
		{name: "oauth", evidence: "auth_required"},
		{name: "stdio", evidence: "unsupported_transport"},
	} {
		t.Run(test.name, func(t *testing.T) {
			stdout, stderr, err := executeMCPCommandForTest(NewMCPCommand(opts), "test", test.name, "--json")
			if err == nil {
				t.Fatalf("test succeeded: %s", stdout)
			}
			var report struct {
				Evidence string             `json:"evidence"`
				Tools    []mcpProbeToolJSON `json:"tools"`
			}
			if jsonErr := json.Unmarshal([]byte(stdout), &report); jsonErr != nil || report.Evidence != test.evidence || report.Tools == nil {
				t.Fatalf("report=%+v jsonErr=%v stdout=%s stderr=%s", report, jsonErr, stdout, stderr)
			}
			combined := stdout + stderr + err.Error()
			for _, forbidden := range []string{"disabled.example", "oauth.example", "/private", "secret-arg", configPath} {
				if strings.Contains(combined, forbidden) {
					t.Fatalf("output leaked %q: %s", forbidden, combined)
				}
			}
		})
	}
	if calls != 0 {
		t.Fatalf("connector calls = %d, want 0", calls)
	}
	after, _ := os.ReadFile(configPath)
	if string(after) != string(before) {
		t.Fatalf("pre-network gates changed config:\n%s", after)
	}

	stdout, _, err := executeMCPCommandForTest(NewMCPCommand(opts), "test", "stdio", "--timeout", "6m", "--json")
	if err == nil || !strings.Contains(stdout, `"evidence": "invalid_input"`) || calls != 0 {
		t.Fatalf("oversized timeout err=%v calls=%d stdout=%s", err, calls, stdout)
	}
}

func TestMCPTestFailuresAreTimedAndRedacted(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := configwriter.WriteTOMLAtomic(configPath, map[string]any{
		"mcp_servers": map[string]any{
			"timeout": map[string]any{"url": "https://timeout.example/private", "enabled": true},
			"connect": map[string]any{"url": "https://connect.example/private", "enabled": true},
			"close":   map[string]any{"url": "https://close.example/private", "enabled": true},
		},
	}); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	before, _ := os.ReadFile(configPath)
	opts := testMCPCommandOptions()
	opts.MCPConfigPath = func() string { return configPath }
	opts.MCPProbeConnector = func(ctx context.Context, def mcpconfig.MCPServerDefinition) (mcpprobe.Session, error) {
		switch def.Name {
		case "timeout":
			<-ctx.Done()
			return nil, ctx.Err()
		case "connect":
			return nil, errors.New("private transport response token-secret-value")
		case "close":
			return &testMCPProbeSession{closeErr: errors.New("private close token-secret-value")}, nil
		default:
			return nil, errors.New("unexpected")
		}
	}
	for _, test := range []struct {
		name     string
		args     []string
		evidence string
	}{
		{name: "timeout", args: []string{"--timeout", "20ms"}, evidence: "timeout"},
		{name: "connect", evidence: "connection_failed"},
		{name: "close", evidence: "connection_failed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			args := append([]string{"test", test.name}, test.args...)
			args = append(args, "--json")
			stdout, stderr, err := executeMCPCommandForTest(NewMCPCommand(opts), args...)
			if err == nil {
				t.Fatalf("test succeeded: %s", stdout)
			}
			var report struct {
				Evidence string `json:"evidence"`
			}
			if jsonErr := json.Unmarshal([]byte(stdout), &report); jsonErr != nil || report.Evidence != test.evidence {
				t.Fatalf("report=%+v jsonErr=%v stdout=%s stderr=%s", report, jsonErr, stdout, stderr)
			}
			combined := stdout + stderr + err.Error()
			for _, forbidden := range []string{"timeout.example", "connect.example", "close.example", "/private", "token-secret-value", configPath} {
				if strings.Contains(combined, forbidden) {
					t.Fatalf("failure leaked %q: %s", forbidden, combined)
				}
			}
		})
	}
	after, _ := os.ReadFile(configPath)
	if string(after) != string(before) {
		t.Fatalf("failed probes changed config:\n%s", after)
	}
}

func TestMCPTestUsesConfiguredTimeoutAndCapsRenderedTools(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := configwriter.WriteTOMLAtomic(configPath, map[string]any{
		"mcp_servers": map[string]any{"many": map[string]any{"url": "https://many.example/mcp", "connect_timeout": "2s", "enabled": true}},
	}); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	tools := make([]descriptor.RawTool, 101)
	for i := range tools {
		tools[i] = descriptor.RawTool{Name: fmt.Sprintf("tool-%03d", i), Description: "safe"}
	}
	opts := testMCPCommandOptions()
	opts.MCPConfigPath = func() string { return configPath }
	opts.MCPProbeConnector = func(ctx context.Context, _ mcpconfig.MCPServerDefinition) (mcpprobe.Session, error) {
		deadline, ok := ctx.Deadline()
		remaining := time.Until(deadline)
		if !ok || remaining < time.Second || remaining > 3*time.Second {
			t.Fatalf("configured timeout remaining = %v, ok=%v", remaining, ok)
		}
		return &testMCPProbeSession{tools: tools}, nil
	}
	stdout, stderr, err := executeMCPCommandForTest(NewMCPCommand(opts), "test", "many", "--json")
	if err != nil {
		t.Fatalf("test many: %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	var report mcpTestReportJSON
	if jsonErr := json.Unmarshal([]byte(stdout), &report); jsonErr != nil {
		t.Fatalf("JSON invalid: %v\n%s", jsonErr, stdout)
	}
	if report.ToolsCount != 101 || len(report.Tools) != 100 || !report.Truncated || report.Tools[0].Name != "tool-000" || report.Tools[99].Name != "tool-099" {
		t.Fatalf("report = %+v", report)
	}
	if strings.Contains(stdout+stderr, "many.example") {
		t.Fatalf("output leaked endpoint: %s %s", stdout, stderr)
	}
}

func TestMCPTestMalformedConfigFailsBeforeConnector(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "private", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatal(err)
	}
	original := []byte("mcp_servers = 'private-secret'\n")
	if err := os.WriteFile(configPath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	opts := testMCPCommandOptions()
	opts.MCPConfigPath = func() string { return configPath }
	opts.MCPProbeConnector = func(context.Context, mcpconfig.MCPServerDefinition) (mcpprobe.Session, error) {
		calls++
		return nil, nil
	}
	stdout, stderr, err := executeMCPCommandForTest(NewMCPCommand(opts), "test", "ink", "--json")
	if err == nil || calls != 0 || !strings.Contains(stdout, `"evidence": "config_rejected"`) {
		t.Fatalf("err=%v calls=%d stdout=%s stderr=%s", err, calls, stdout, stderr)
	}
	combined := stdout + stderr + err.Error()
	if strings.Contains(combined, "private-secret") || strings.Contains(combined, configPath) {
		t.Fatalf("malformed config leaked: %s", combined)
	}
	after, _ := os.ReadFile(configPath)
	if string(after) != string(original) {
		t.Fatalf("malformed test changed config: %s", after)
	}
}

func TestMCPConfigureIncludePersistsCanonicalSelectionWithoutLeaks(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "profile", "config.toml")
	if err := configwriter.WriteTOMLAtomic(configPath, map[string]any{
		"hermes": map[string]any{"model": "keep-me"},
		"mcp_servers": map[string]any{
			"demo": map[string]any{
				"url":             "https://example.test/private?token=query-secret",
				"headers":         map[string]any{"Authorization": "Bearer header-secret"},
				"enabled":         true,
				"connect_timeout": "3s",
				"tools":           map[string]any{"exclude": []string{"old"}, "prompts": false},
			},
			"sibling": map[string]any{"url": "https://sibling.example/mcp", "enabled": true},
		},
	}); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	opts := testMCPCommandOptions()
	opts.MCPConfigPath = func() string { return configPath }
	stdout, stderr, err := executeMCPCommandForTest(NewMCPCommand(opts), "configure", "demo", "--include", "search,create", "--json")
	if err != nil {
		t.Fatalf("configure: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var report struct {
		Build                  BuildProvenance `json:"build"`
		Action                 string          `json:"action"`
		Name                   string          `json:"name"`
		Evidence               string          `json:"evidence"`
		SelectionMode          string          `json:"selection_mode"`
		SelectedCount          int             `json:"selected_count"`
		RuntimeRefreshRequired bool            `json:"runtime_refresh_required"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &report); jsonErr != nil {
		t.Fatalf("JSON invalid: %v\n%s", jsonErr, stdout)
	}
	if report.Build.Version != testMCPVersion || report.Action != "mcp_configure" || report.Name != "demo" || report.Evidence != "configured" || report.SelectionMode != "include" || report.SelectedCount != 2 || !report.RuntimeRefreshRequired {
		t.Fatalf("report=%+v", report)
	}
	combined := stdout + stderr
	for _, forbidden := range []string{"search", "create", "private", "query-secret", "header-secret", configPath} {
		if strings.Contains(combined, forbidden) {
			t.Fatalf("configure output leaked %q: %s", forbidden, combined)
		}
	}
	doc, err := configwriter.ReadTOMLDoc(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	servers, _ := doc["mcp_servers"].(map[string]any)
	demo, _ := servers["demo"].(map[string]any)
	tools, _ := demo["tools"].(map[string]any)
	include, _ := tools["include"].([]any)
	if len(include) != 2 || include[0] != "create" || include[1] != "search" {
		t.Fatalf("include=%#v", include)
	}
	if _, exists := tools["exclude"]; exists || tools["prompts"] != false {
		t.Fatalf("tools=%#v", tools)
	}
	if demo["url"] != "https://example.test/private?token=query-secret" || demo["connect_timeout"] != "3s" || demo["enabled"] != true {
		t.Fatalf("demo=%#v", demo)
	}
	if _, exists := servers["sibling"]; !exists {
		t.Fatalf("sibling lost: %#v", servers)
	}
	hermes, _ := doc["hermes"].(map[string]any)
	if hermes["model"] != "keep-me" {
		t.Fatalf("unrelated config changed: %#v", doc)
	}

	registry := toolkit.NewRegistry()
	reportRuntime := mcpruntime.RegisterConfiguredHTTP(context.Background(), registry, servers, remote.Connector(func(context.Context, mcpconfig.MCPServerDefinition) (remote.Session, error) {
		return &testMCPConfigureRuntimeSession{tools: []descriptor.RawTool{
			{Name: "search", InputSchema: json.RawMessage(`{"type":"object"}`)},
			{Name: "ignored", InputSchema: json.RawMessage(`{"type":"object"}`)},
			{Name: "create", InputSchema: json.RawMessage(`{"type":"object"}`)},
		}}, nil
	}), mcpruntime.Options{})
	if _, ok := registry.Get("mcp__demo__create"); !ok {
		t.Fatalf("configured create missing: report=%+v", reportRuntime)
	}
	if _, ok := registry.Get("mcp__demo__search"); !ok {
		t.Fatalf("configured search missing: report=%+v", reportRuntime)
	}
	if _, ok := registry.Get("mcp__demo__ignored"); ok {
		t.Fatalf("unselected tool registered: report=%+v", reportRuntime)
	}
}

func TestMCPConfigureAliasNoneAndAllSelection(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := configwriter.WriteTOMLAtomic(configPath, map[string]any{
		"mcp_servers": map[string]any{
			"demo":  map[string]any{"url": "https://example.test/private", "enabled": true, "tools": map[string]any{"include": []string{"old"}, "exclude": []string{"other"}, "prompts": false}},
			"plain": map[string]any{"url": "https://plain.example/mcp", "enabled": true, "tools": map[string]any{"include": []string{"old"}}},
		},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	opts := testMCPCommandOptions()
	opts.MCPConfigPath = func() string { return configPath }
	stdout, stderr, err := executeMCPCommandForTest(NewMCPCommand(opts), "config", "demo", "--none", "--json")
	if err != nil {
		t.Fatalf("config alias none: %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	var report mcpConfigureReportJSON
	if jsonErr := json.Unmarshal([]byte(stdout), &report); jsonErr != nil || report.Evidence != "configured" || report.SelectionMode != "none" || report.SelectedCount != 0 || !report.RuntimeRefreshRequired {
		t.Fatalf("none report=%+v err=%v stdout=%s", report, jsonErr, stdout)
	}
	doc, _ := configwriter.ReadTOMLDoc(configPath)
	servers, _ := doc["mcp_servers"].(map[string]any)
	demo, _ := servers["demo"].(map[string]any)
	tools, _ := demo["tools"].(map[string]any)
	include, exists := tools["include"].([]any)
	if !exists || len(include) != 0 || tools["prompts"] != false {
		t.Fatalf("none tools=%#v", tools)
	}
	if _, exists := tools["exclude"]; exists {
		t.Fatalf("none retained exclude: %#v", tools)
	}

	stdout, stderr, err = executeMCPCommandForTest(NewMCPCommand(opts), "configure", "demo", "--all", "--json")
	if err != nil {
		t.Fatalf("all: %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	doc, _ = configwriter.ReadTOMLDoc(configPath)
	servers, _ = doc["mcp_servers"].(map[string]any)
	demo, _ = servers["demo"].(map[string]any)
	tools, _ = demo["tools"].(map[string]any)
	if tools["prompts"] != false {
		t.Fatalf("all lost non-selection tools config: %#v", tools)
	}
	if _, exists := tools["include"]; exists {
		t.Fatalf("all retained include: %#v", tools)
	}
	if _, exists := tools["exclude"]; exists {
		t.Fatalf("all retained exclude: %#v", tools)
	}

	stdout, stderr, err = executeMCPCommandForTest(NewMCPCommand(opts), "configure", "plain", "--all")
	if err != nil || !strings.Contains(stdout, "Reload or restart") {
		t.Fatalf("plain all: %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	doc, _ = configwriter.ReadTOMLDoc(configPath)
	servers, _ = doc["mcp_servers"].(map[string]any)
	plain, _ := servers["plain"].(map[string]any)
	if _, exists := plain["tools"]; exists {
		t.Fatalf("empty tools table survived: %#v", plain)
	}
	if strings.Contains(stdout+stderr, "plain.example") || strings.Contains(stdout+stderr, configPath) {
		t.Fatalf("all output leaked config: %s %s", stdout, stderr)
	}
}

func TestMCPConfigureFailuresAreRedactedAndNonMutating(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "private-profile", "config.toml")
	if err := configwriter.WriteTOMLAtomic(configPath, map[string]any{
		"mcp_servers": map[string]any{
			"demo":      map[string]any{"url": "https://example.test/private?token=query-secret", "headers": map[string]any{"Authorization": "Bearer header-secret"}, "enabled": true},
			"bad_tools": map[string]any{"url": "https://bad.example/mcp", "enabled": true, "tools": "private-tools-secret"},
		},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	opts := testMCPCommandOptions()
	opts.MCPConfigPath = func() string { return configPath }
	tooMany := make([]string, 101)
	for i := range tooMany {
		tooMany[i] = fmt.Sprintf("tool_%03d", i)
	}
	tests := []struct {
		name     string
		args     []string
		evidence string
	}{
		{name: "no mode", args: []string{"configure", "demo", "--json"}, evidence: "invalid_input"},
		{name: "conflicting modes", args: []string{"configure", "demo", "--none", "--all", "--json"}, evidence: "invalid_input"},
		{name: "missing", args: []string{"configure", "missing", "--all", "--json"}, evidence: "not_found"},
		{name: "invalid server", args: []string{"configure", "bad/name", "--all", "--json"}, evidence: "invalid_input"},
		{name: "empty include", args: []string{"configure", "demo", "--include", "", "--json"}, evidence: "invalid_input"},
		{name: "duplicate include", args: []string{"configure", "demo", "--include", "same,same", "--json"}, evidence: "invalid_input"},
		{name: "long include", args: []string{"configure", "demo", "--include", strings.Repeat("x", 129), "--json"}, evidence: "invalid_input"},
		{name: "control include", args: []string{"configure", "demo", "--include", "bad\nname", "--json"}, evidence: "invalid_input"},
		{name: "too many", args: []string{"configure", "demo", "--include", strings.Join(tooMany, ","), "--json"}, evidence: "invalid_input"},
		{name: "malformed tools", args: []string{"configure", "bad_tools", "--all", "--json"}, evidence: "config_rejected"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before, _ := os.ReadFile(configPath)
			stdout, stderr, err := executeMCPCommandForTest(NewMCPCommand(opts), test.args...)
			if err == nil {
				t.Fatalf("command succeeded: %s", stdout)
			}
			var report mcpConfigureReportJSON
			if jsonErr := json.Unmarshal([]byte(stdout), &report); jsonErr != nil || report.Evidence != test.evidence || report.RuntimeRefreshRequired {
				t.Fatalf("report=%+v jsonErr=%v stdout=%s stderr=%s err=%v", report, jsonErr, stdout, stderr, err)
			}
			after, _ := os.ReadFile(configPath)
			if string(after) != string(before) {
				t.Fatalf("failure changed config:\n%s", after)
			}
			combined := stdout + stderr + err.Error()
			for _, forbidden := range []string{"query-secret", "header-secret", "private-tools-secret", "example.test", configPath, "bad\\nname"} {
				if strings.Contains(combined, forbidden) {
					t.Fatalf("failure leaked %q: %s", forbidden, combined)
				}
			}
		})
	}
}

func TestMCPConfigureMalformedDocumentIsNonMutating(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "private-profile", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatal(err)
	}
	original := []byte("mcp_servers = 'private-document-secret'\n")
	if err := os.WriteFile(configPath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	opts := testMCPCommandOptions()
	opts.MCPConfigPath = func() string { return configPath }
	stdout, stderr, err := executeMCPCommandForTest(NewMCPCommand(opts), "configure", "demo", "--all", "--json")
	if err == nil {
		t.Fatalf("malformed configure succeeded: %s", stdout)
	}
	var report mcpConfigureReportJSON
	if jsonErr := json.Unmarshal([]byte(stdout), &report); jsonErr != nil || report.Evidence != "config_rejected" {
		t.Fatalf("report=%+v err=%v stdout=%s stderr=%s", report, jsonErr, stdout, stderr)
	}
	after, _ := os.ReadFile(configPath)
	if string(after) != string(original) {
		t.Fatalf("malformed config changed: %s", after)
	}
	combined := stdout + stderr + err.Error()
	if strings.Contains(combined, "private-document-secret") || strings.Contains(combined, configPath) {
		t.Fatalf("malformed config leaked: %s", combined)
	}
}

func TestMCPAddCustomHTTPPersistsWithoutLeakingEndpointDetails(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "profile", "config.toml")
	opts := testMCPCommandOptions()
	opts.MCPConfigPath = func() string { return configPath }

	stdout, stderr, err := executeMCPCommandForTest(
		NewMCPCommand(opts),
		"add", "custom",
		"--url", "https://example.test/private?token=leak",
		"--auth", "oauth",
		"--json",
	)
	if err != nil {
		t.Fatalf("Execute add: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var added struct {
		Build     BuildProvenance `json:"build"`
		Action    string          `json:"action"`
		Name      string          `json:"name"`
		Evidence  string          `json:"evidence"`
		Transport string          `json:"transport"`
		Auth      string          `json:"auth"`
		Target    string          `json:"target"`
	}
	if err := json.Unmarshal([]byte(stdout), &added); err != nil {
		t.Fatalf("add JSON invalid: %v\n%s", err, stdout)
	}
	if added.Build.Version != testMCPVersion || added.Action != "mcp_add" || added.Name != "custom" || added.Evidence != "configured_unverified" || added.Transport != "http" || added.Auth != "oauth" || added.Target != "https://example.test" {
		t.Fatalf("added = %+v", added)
	}
	if strings.Contains(stdout+stderr, "private") || strings.Contains(stdout+stderr, "token=leak") {
		t.Fatalf("add output leaked endpoint details: stdout=%s stderr=%s", stdout, stderr)
	}
	doc, err := configwriter.ReadTOMLDoc(configPath)
	if err != nil {
		t.Fatalf("ReadTOMLDoc: %v", err)
	}
	servers, _ := doc["mcp_servers"].(map[string]any)
	custom, _ := servers["custom"].(map[string]any)
	if custom["url"] != "https://example.test/private?token=leak" || custom["auth"] != "oauth" || custom["enabled"] != true {
		t.Fatalf("custom config = %#v", custom)
	}

	stdout, stderr, err = executeMCPCommandForTest(NewMCPCommand(opts), "list", "--json")
	if err != nil {
		t.Fatalf("Execute list: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var listed struct {
		Action  string `json:"action"`
		Count   int    `json:"count"`
		Entries []struct {
			Name      string `json:"name"`
			Transport string `json:"transport"`
			Enabled   bool   `json:"enabled"`
			Auth      string `json:"auth"`
			Target    string `json:"target"`
		} `json:"entries"`
	}
	if err := json.Unmarshal([]byte(stdout), &listed); err != nil {
		t.Fatalf("list JSON invalid: %v\n%s", err, stdout)
	}
	if listed.Action != "mcp_list" || listed.Count != 1 || len(listed.Entries) != 1 || listed.Entries[0].Name != "custom" || listed.Entries[0].Target != "https://example.test" || !listed.Entries[0].Enabled || listed.Entries[0].Auth != "oauth" {
		t.Fatalf("listed = %+v", listed)
	}
	if strings.Contains(stdout+stderr, "private") || strings.Contains(stdout+stderr, "token=leak") {
		t.Fatalf("list output leaked endpoint details: stdout=%s stderr=%s", stdout, stderr)
	}
}

func TestMCPAddDuplicateRequiresForceAndPreservesTools(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := configwriter.WriteTOMLAtomic(configPath, map[string]any{
		"mcp_servers": map[string]any{
			"custom": map[string]any{
				"url":     "https://old.example/mcp",
				"enabled": true,
				"tools":   map[string]any{"include": []string{"one", "two"}},
			},
		},
	}); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	before, _ := os.ReadFile(configPath)
	opts := testMCPCommandOptions()
	opts.MCPConfigPath = func() string { return configPath }
	stdout, _, err := executeMCPCommandForTest(NewMCPCommand(opts), "add", "custom", "--url", "https://new.example/private", "--json")
	if err == nil {
		t.Fatalf("duplicate succeeded: %s", stdout)
	}
	var rejected struct {
		Evidence string `json:"evidence"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &rejected); jsonErr != nil || rejected.Evidence != "already_exists" {
		t.Fatalf("rejected=%+v jsonErr=%v stdout=%s", rejected, jsonErr, stdout)
	}
	after, _ := os.ReadFile(configPath)
	if string(after) != string(before) {
		t.Fatalf("duplicate changed config:\n%s", after)
	}
	stdout, stderr, err := executeMCPCommandForTest(NewMCPCommand(opts), "add", "custom", "--url", "https://new.example/private", "--auth", "oauth", "--force", "--json")
	if err != nil {
		t.Fatalf("force add: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	doc, _ := configwriter.ReadTOMLDoc(configPath)
	servers, _ := doc["mcp_servers"].(map[string]any)
	custom, _ := servers["custom"].(map[string]any)
	tools, _ := custom["tools"].(map[string]any)
	include, _ := tools["include"].([]any)
	if custom["url"] != "https://new.example/private" || custom["auth"] != "oauth" || len(include) != 2 {
		t.Fatalf("custom = %#v", custom)
	}
}

func TestMCPAddRejectsUnsupportedInputsWithoutWriting(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing URL", args: []string{"add", "custom", "--json"}},
		{name: "userinfo", args: []string{"add", "custom", "--url", "https://user:secret@example.test/mcp", "--json"}},
		{name: "wrong scheme", args: []string{"add", "custom", "--url", "file:///tmp/server", "--json"}},
		{name: "header auth", args: []string{"add", "custom", "--url", "https://example.test/mcp", "--auth", "header", "--json"}},
		{name: "stdio command", args: []string{"add", "custom", "--command", "npx", "--json"}},
		{name: "env", args: []string{"add", "custom", "--url", "https://example.test/mcp", "--env", "TOKEN=secret", "--json"}},
		{name: "probe timeout", args: []string{"add", "custom", "--url", "https://example.test/mcp", "--connect-timeout", "0", "--json"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "config.toml")
			opts := testMCPCommandOptions()
			opts.MCPConfigPath = func() string { return configPath }
			stdout, stderr, err := executeMCPCommandForTest(NewMCPCommand(opts), tc.args...)
			if err == nil {
				t.Fatalf("command succeeded: stdout=%s stderr=%s", stdout, stderr)
			}
			var report struct {
				Evidence string `json:"evidence"`
			}
			if jsonErr := json.Unmarshal([]byte(stdout), &report); jsonErr != nil || report.Evidence != "invalid_input" {
				t.Fatalf("report=%+v jsonErr=%v stdout=%s stderr=%s err=%v", report, jsonErr, stdout, stderr, err)
			}
			if strings.Contains(stdout+stderr+err.Error(), "secret") {
				t.Fatalf("output leaked input: stdout=%s stderr=%s err=%v", stdout, stderr, err)
			}
			if _, statErr := os.Stat(configPath); !os.IsNotExist(statErr) {
				t.Fatalf("config stat err = %v, want not-exist", statErr)
			}
		})
	}
}

func TestMCPRemoveDeletesExactConfigAndPreservesCredentialsArtifacts(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := configwriter.WriteTOMLAtomic(configPath, map[string]any{
		"hermes": map[string]any{"model": "keep-me"},
		"mcp_servers": map[string]any{
			"custom": map[string]any{
				"url":     "https://example.test/private?token=must-not-leak",
				"headers": map[string]string{"Authorization": "Bearer header-secret"},
				"enabled": true,
			},
			"sibling": map[string]any{"url": "https://sibling.example/mcp", "enabled": true},
		},
	}); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	opts := testMCPCommandOptions()
	opts.MCPConfigPath = func() string { return configPath }
	stdout, stderr, err := executeMCPCommandForTest(NewMCPCommand(opts), "remove", "custom", "--yes", "--json")
	if err != nil {
		t.Fatalf("remove: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var report struct {
		Build                BuildProvenance `json:"build"`
		Action               string          `json:"action"`
		Name                 string          `json:"name"`
		Evidence             string          `json:"evidence"`
		CredentialsPreserved bool            `json:"credentials_preserved"`
		ArtifactsPreserved   bool            `json:"artifacts_preserved"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &report); jsonErr != nil {
		t.Fatalf("JSON invalid: %v\n%s", jsonErr, stdout)
	}
	if report.Build.Version != testMCPVersion || report.Action != "mcp_remove" || report.Name != "custom" || report.Evidence != "removed" || !report.CredentialsPreserved || !report.ArtifactsPreserved {
		t.Fatalf("report = %+v", report)
	}
	combined := stdout + stderr
	for _, secret := range []string{"private", "must-not-leak", "header-secret", configPath} {
		if strings.Contains(combined, secret) {
			t.Fatalf("remove output leaked %q: %s", secret, combined)
		}
	}
	doc, err := configwriter.ReadTOMLDoc(configPath)
	if err != nil {
		t.Fatalf("ReadTOMLDoc: %v", err)
	}
	servers, _ := doc["mcp_servers"].(map[string]any)
	if _, exists := servers["custom"]; exists {
		t.Fatalf("custom survived: %#v", servers)
	}
	if _, exists := servers["sibling"]; !exists {
		t.Fatalf("sibling removed: %#v", servers)
	}
	hermes, _ := doc["hermes"].(map[string]any)
	if hermes["model"] != "keep-me" {
		t.Fatalf("unrelated config changed: %#v", hermes)
	}
}

func TestMCPRemoveRequiresYesAndMissingIsNonMutating(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := configwriter.WriteTOMLAtomic(configPath, map[string]any{
		"mcp_servers": map[string]any{
			"custom": map[string]any{"url": "https://example.test/mcp", "enabled": true},
		},
	}); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	before, _ := os.ReadFile(configPath)
	opts := testMCPCommandOptions()
	opts.MCPConfigPath = func() string { return configPath }
	stdout, stderr, err := executeMCPCommandForTest(NewMCPCommand(opts), "remove", "custom", "--json")
	if err == nil {
		t.Fatalf("remove without --yes succeeded: %s", stdout)
	}
	var report struct {
		Evidence             string `json:"evidence"`
		CredentialsPreserved bool   `json:"credentials_preserved"`
		ArtifactsPreserved   bool   `json:"artifacts_preserved"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &report); jsonErr != nil || report.Evidence != "confirmation_required" || !report.CredentialsPreserved || !report.ArtifactsPreserved {
		t.Fatalf("report=%+v jsonErr=%v stdout=%s stderr=%s", report, jsonErr, stdout, stderr)
	}
	after, _ := os.ReadFile(configPath)
	if string(after) != string(before) {
		t.Fatalf("unconfirmed remove changed config:\n%s", after)
	}
	stdout, stderr, err = executeMCPCommandForTest(NewMCPCommand(opts), "remove", "missing", "--yes", "--json")
	if err == nil {
		t.Fatalf("remove missing succeeded: %s", stdout)
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &report); jsonErr != nil || report.Evidence != "not_found" {
		t.Fatalf("missing report=%+v jsonErr=%v stdout=%s stderr=%s", report, jsonErr, stdout, stderr)
	}
	after, _ = os.ReadFile(configPath)
	if string(after) != string(before) {
		t.Fatalf("missing remove changed config:\n%s", after)
	}
	stdout, stderr, err = executeMCPCommandForTest(NewMCPCommand(opts), "remove", "bad/name", "--yes", "--json")
	if err == nil {
		t.Fatalf("invalid-name remove succeeded: %s", stdout)
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &report); jsonErr != nil || report.Evidence != "invalid_input" {
		t.Fatalf("invalid report=%+v jsonErr=%v stdout=%s stderr=%s", report, jsonErr, stdout, stderr)
	}
	after, _ = os.ReadFile(configPath)
	if string(after) != string(before) {
		t.Fatalf("invalid remove changed config:\n%s", after)
	}
}

func TestMCPRemoveAliasAndMalformedConfig(t *testing.T) {
	t.Run("rm alias", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.toml")
		if err := configwriter.WriteTOMLAtomic(configPath, map[string]any{
			"mcp_servers": map[string]any{"custom": map[string]any{"url": "https://example.test/mcp", "enabled": true}},
		}); err != nil {
			t.Fatalf("seed: %v", err)
		}
		opts := testMCPCommandOptions()
		opts.MCPConfigPath = func() string { return configPath }
		stdout, stderr, err := executeMCPCommandForTest(NewMCPCommand(opts), "rm", "custom", "-y")
		if err != nil || !strings.Contains(stdout, "Credentials and artifacts were preserved") {
			t.Fatalf("rm alias: %v stdout=%s stderr=%s", err, stdout, stderr)
		}
	})
	t.Run("malformed", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "private-profile", "config.toml")
		if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		original := []byte("mcp_servers = 'private-secret'\n")
		if err := os.WriteFile(configPath, original, 0o600); err != nil {
			t.Fatalf("seed: %v", err)
		}
		opts := testMCPCommandOptions()
		opts.MCPConfigPath = func() string { return configPath }
		stdout, stderr, err := executeMCPCommandForTest(NewMCPCommand(opts), "remove", "custom", "--yes", "--json")
		if err == nil {
			t.Fatalf("malformed remove succeeded: %s", stdout)
		}
		var report struct {
			Evidence string `json:"evidence"`
		}
		if jsonErr := json.Unmarshal([]byte(stdout), &report); jsonErr != nil || report.Evidence != "config_rejected" {
			t.Fatalf("report=%+v jsonErr=%v stdout=%s", report, jsonErr, stdout)
		}
		combined := stdout + stderr + err.Error()
		if strings.Contains(combined, configPath) || strings.Contains(combined, "private-secret") {
			t.Fatalf("remove leaked config detail: %s", combined)
		}
		after, _ := os.ReadFile(configPath)
		if string(after) != string(original) {
			t.Fatalf("malformed remove changed config:\n%s", after)
		}
	})
}

func TestMCPListRedactsExistingHTTPAndStdioSecrets(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := configwriter.WriteTOMLAtomic(configPath, map[string]any{
		"mcp_servers": map[string]any{
			"zeta": map[string]any{
				"command": `/private/bin/stdio-server`,
				"args":    []string{"--token", "argument-secret"},
				"env":     map[string]string{"PRIVATE_TOKEN": "environment-secret"},
				"enabled": false,
			},
			"alpha": map[string]any{
				"url":     "https://user:password@example.test/private/path?token=query-secret#fragment",
				"headers": map[string]string{"Authorization": "Bearer header-secret"},
				"enabled": true,
				"tools":   map[string]any{"include": []string{"one", "two"}},
			},
		},
	}); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	opts := testMCPCommandOptions()
	opts.MCPConfigPath = func() string { return configPath }
	stdout, stderr, err := executeMCPCommandForTest(NewMCPCommand(opts), "list", "--json")
	if err != nil {
		t.Fatalf("list: %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	var report struct {
		Entries []struct {
			Name          string `json:"name"`
			Target        string `json:"target"`
			Auth          string `json:"auth"`
			ToolsSelected int    `json:"tools_selected"`
		} `json:"entries"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &report); jsonErr != nil {
		t.Fatalf("JSON invalid: %v\n%s", jsonErr, stdout)
	}
	if len(report.Entries) != 2 || report.Entries[0].Name != "alpha" || report.Entries[0].Target != "https://example.test" || report.Entries[0].Auth != "header" || report.Entries[0].ToolsSelected != 2 || report.Entries[1].Name != "zeta" || report.Entries[1].Target != "stdio-server" {
		t.Fatalf("entries = %+v", report.Entries)
	}
	combined := stdout + stderr
	for _, secret := range []string{"user", "password", "private/path", "query-secret", "fragment", "argument-secret", "PRIVATE_TOKEN", "environment-secret", "header-secret", "/private/bin"} {
		if strings.Contains(combined, secret) {
			t.Fatalf("output leaked %q: %s", secret, combined)
		}
	}
}

func TestMCPListEmptyAndMalformedConfig(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.toml")
		opts := testMCPCommandOptions()
		opts.MCPConfigPath = func() string { return configPath }
		stdout, stderr, err := executeMCPCommandForTest(NewMCPCommand(opts), "ls", "--json")
		if err != nil {
			t.Fatalf("list empty: %v stdout=%s stderr=%s", err, stdout, stderr)
		}
		var report struct {
			Evidence string `json:"evidence"`
			Count    int    `json:"count"`
			Entries  []any  `json:"entries"`
		}
		if jsonErr := json.Unmarshal([]byte(stdout), &report); jsonErr != nil || report.Evidence != "listed" || report.Count != 0 || len(report.Entries) != 0 {
			t.Fatalf("report=%+v jsonErr=%v stdout=%s", report, jsonErr, stdout)
		}
	})
	t.Run("malformed", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "private-profile", "config.toml")
		if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(configPath, []byte("mcp_servers = 'private-secret'\n"), 0o600); err != nil {
			t.Fatalf("seed: %v", err)
		}
		opts := testMCPCommandOptions()
		opts.MCPConfigPath = func() string { return configPath }
		stdout, stderr, err := executeMCPCommandForTest(NewMCPCommand(opts), "list", "--json")
		if err == nil {
			t.Fatalf("malformed list succeeded: %s", stdout)
		}
		var report struct {
			Evidence string `json:"evidence"`
			Count    int    `json:"count"`
			Entries  []any  `json:"entries"`
		}
		if jsonErr := json.Unmarshal([]byte(stdout), &report); jsonErr != nil || report.Evidence != "config_rejected" || report.Count != 0 || len(report.Entries) != 0 {
			t.Fatalf("report=%+v jsonErr=%v stdout=%s", report, jsonErr, stdout)
		}
		combined := stdout + stderr + err.Error()
		if strings.Contains(combined, configPath) || strings.Contains(combined, "private-secret") {
			t.Fatalf("output leaked config details: %s", combined)
		}
	})
}
