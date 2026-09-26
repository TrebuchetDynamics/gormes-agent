package gormescli

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/configwriter"
	mcpconfig "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/descriptor"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/remote"
	mcpruntime "github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/runtime"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit"
)

func TestMCPEnableDisableListResolverAndFreshRegistryTransitions(t *testing.T) {
	tests := []struct {
		name      string
		transport string
		server    map[string]any
	}{
		{
			name:      "http_fixture",
			transport: "http",
			server: map[string]any{
				"url":     "https://example.test/mcp?token=query-secret",
				"headers": map[string]any{"Authorization": "Bearer header-secret"},
			},
		},
		{
			name:      "stdio_fixture",
			transport: "stdio",
			server: map[string]any{
				"command": "/private/bin/mcp-fixture",
				"args":    []string{"--mode", "fixture"},
				"env":     map[string]any{"MCP_TOKEN": "stdio-secret"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "profile", "config.toml")
			server := map[string]any{
				"enabled":         true,
				"auth":            "none",
				"timeout":         "9s",
				"connect_timeout": "3s",
				"sampling": map[string]any{
					"enabled": true, "model": "fixture-model", "max_tokens_cap": 16,
					"max_rpm": 3, "allowed_models": []string{"fixture-model"},
					"max_tool_rounds": 2, "log_level": "warn",
				},
				"tools": map[string]any{
					"include": []string{"ping"}, "exclude": []string{"private"}, "prompts": false,
				},
			}
			for key, value := range test.server {
				server[key] = value
			}
			if err := configwriter.WriteTOMLAtomic(configPath, map[string]any{
				"mcp_servers": map[string]any{test.name: server},
			}); err != nil {
				t.Fatalf("seed config: %v", err)
			}
			opts := testMCPCommandOptions()
			opts.MCPConfigPath = func() string { return configPath }

			for _, enabled := range []bool{false, true} {
				verb, evidence := "disable", "disabled"
				if enabled {
					verb, evidence = "enable", "enabled"
				}
				stdout, stderr, err := executeMCPCommandForTest(NewMCPCommand(opts), verb, test.name, "--json")
				if err != nil {
					t.Fatalf("mcp %s: %v stdout=%s stderr=%s", verb, err, stdout, stderr)
				}
				var toggle struct {
					Name                   string `json:"name"`
					Evidence               string `json:"evidence"`
					Enabled                bool   `json:"enabled"`
					RuntimeRefreshRequired bool   `json:"runtime_refresh_required"`
				}
				if jsonErr := json.Unmarshal([]byte(stdout), &toggle); jsonErr != nil || toggle.Name != test.name || toggle.Evidence != evidence || toggle.Enabled != enabled || !toggle.RuntimeRefreshRequired {
					t.Fatalf("toggle=%+v jsonErr=%v stdout=%s", toggle, jsonErr, stdout)
				}
				for _, secret := range []string{"query-secret", "header-secret", "stdio-secret", "/private/bin", configPath} {
					if strings.Contains(stdout+stderr, secret) {
						t.Fatalf("toggle output leaked %q: %s %s", secret, stdout, stderr)
					}
				}

				listOut, listErrOut, err := executeMCPCommandForTest(NewMCPCommand(opts), "list", "--json")
				if err != nil {
					t.Fatalf("mcp list: %v stdout=%s stderr=%s", err, listOut, listErrOut)
				}
				var list struct {
					Evidence string `json:"evidence"`
					Count    int    `json:"count"`
					Entries  []struct {
						Name      string `json:"name"`
						Transport string `json:"transport"`
						Enabled   bool   `json:"enabled"`
						Status    string `json:"status"`
					} `json:"entries"`
				}
				if jsonErr := json.Unmarshal([]byte(listOut), &list); jsonErr != nil || list.Evidence != "listed" || list.Count != 1 || len(list.Entries) != 1 || list.Entries[0].Name != test.name || list.Entries[0].Transport != test.transport || list.Entries[0].Enabled != enabled {
					t.Fatalf("list=%+v jsonErr=%v stdout=%s", list, jsonErr, listOut)
				}
				wantListStatus := string(mcpconfig.MCPConfigStatusDisabled)
				if enabled {
					wantListStatus = string(mcpconfig.MCPConfigStatusReady)
				}
				if list.Entries[0].Status != wantListStatus {
					t.Fatalf("list status=%q want=%q", list.Entries[0].Status, wantListStatus)
				}

				doc, err := configwriter.ReadTOMLDoc(configPath)
				if err != nil {
					t.Fatalf("read config: %v", err)
				}
				servers, _ := doc["mcp_servers"].(map[string]any)
				gotServer, _ := servers[test.name].(map[string]any)
				if gotServer["enabled"] != enabled {
					t.Fatalf("persisted enabled=%#v want=%t", gotServer["enabled"], enabled)
				}
				for field, want := range server {
					if field == "enabled" {
						continue
					}
					got, exists := gotServer[field]
					if !exists {
						t.Fatalf("field %q lost after %s: %#v", field, verb, gotServer)
					}
					wantJSON, _ := json.Marshal(want)
					gotJSON, _ := json.Marshal(got)
					if string(gotJSON) != string(wantJSON) {
						t.Fatalf("field %q changed after %s: got=%s want=%s", field, verb, gotJSON, wantJSON)
					}
				}

				resolution, err := mcpconfig.ResolveMCPConfig(doc, mcpconfig.MCPConfigOptions{LookupEnv: func(string) (string, bool) { return "", false }})
				if err != nil {
					t.Fatalf("resolve persisted config: %v", err)
				}
				definition, ok := resolution.Server(test.name)
				wantTransport := mcpconfig.MCPTransport("")
				if enabled {
					wantTransport = mcpconfig.MCPTransport(test.transport)
				}
				if !ok || definition.Enabled != enabled || definition.Transport != wantTransport {
					t.Fatalf("resolved definition=%+v found=%t want enabled=%t transport=%q", definition, ok, enabled, wantTransport)
				}
				status, ok := resolution.Status(test.name)
				wantStatus := mcpconfig.MCPConfigStatusDisabled
				if enabled {
					wantStatus = mcpconfig.MCPConfigStatusReady
				}
				if !ok || status.Status != wantStatus || status.Enabled != enabled {
					t.Fatalf("resolved status=%+v found=%t want status=%q enabled=%t", status, ok, wantStatus, enabled)
				}

				registry := toolkit.NewRegistry()
				connectorCalls := 0
				runtimeReport := mcpruntime.RegisterConfiguredHTTP(context.Background(), registry, servers, remote.Connector(func(_ context.Context, def mcpconfig.MCPServerDefinition) (remote.Session, error) {
					connectorCalls++
					if def.Transport != mcpconfig.MCPTransportHTTP {
						return nil, fmt.Errorf("unexpected transport %q", def.Transport)
					}
					return &testMCPConfigureRuntimeSession{tools: []descriptor.RawTool{{Name: "ping", InputSchema: json.RawMessage(`{"type":"object"}`)}}}, nil
				}), mcpruntime.Options{})
				toolName := "mcp__" + test.name + "__ping"
				_, registered := registry.Get(toolName)
				if !enabled || test.transport != "http" {
					if registered || connectorCalls != 0 {
						t.Fatalf("disabled/stdio server reached fresh HTTP registry: registered=%t connector_calls=%d report=%+v", registered, connectorCalls, runtimeReport)
					}
					if enabled {
						unsupported := false
						for _, row := range runtimeReport.Statuses {
							unsupported = unsupported || (row.Server == test.name && row.Evidence == mcpruntime.EvidenceUnsupported)
						}
						if !unsupported {
							t.Fatalf("enabled stdio config was not explicitly rejected by HTTP registry: %+v", runtimeReport)
						}
					}
				} else if !registered || connectorCalls != 1 {
					t.Fatalf("enabled HTTP server missing from fresh registry: registered=%t connector_calls=%d report=%+v", registered, connectorCalls, runtimeReport)
				}
			}
		})
	}
}
