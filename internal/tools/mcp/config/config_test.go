package config

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseMCPConfigJSONRejectsTrailingDocuments(t *testing.T) {
	_, err := ParseMCPConfigJSON([]byte(`{"mcp_servers": {}} {"mcp_servers": {}}`), MCPConfigOptions{})
	if err == nil {
		t.Fatalf("ParseMCPConfigJSON succeeded for concatenated JSON documents")
	}
	if !strings.Contains(err.Error(), "trailing content") {
		t.Fatalf("error = %q, want trailing content evidence", err.Error())
	}
}

func TestParseMCPBoolTreatsJSONNumbersLikeDecodedFloats(t *testing.T) {
	if got := parseMCPBool(json.Number("0.5"), false); !got {
		t.Fatalf("parseMCPBool(json.Number(0.5), false) = false, want true to match decoded float64 truthiness")
	}
	if got := parseMCPBool(json.Number("0"), true); got {
		t.Fatalf("parseMCPBool(json.Number(0), true) = true, want false")
	}
	if got := parseMCPBool(json.Number("not-a-number"), true); !got {
		t.Fatalf("parseMCPBool(invalid json.Number, true) = false, want fallback true")
	}
}

func TestResolveMCPConfigRejectsAmbiguousTopLevelServerBlocks(t *testing.T) {
	resolved, err := ResolveMCPConfig(map[string]any{
		"mcp_servers": map[string]any{
			"stdio": map[string]any{"command": "npx"},
		},
		"mcpServers": map[string]any{
			"remote": map[string]any{"url": "https://mcp.example.test/sse"},
		},
	}, MCPConfigOptions{LookupEnv: func(string) (string, bool) { return "", false }})
	if err == nil {
		t.Fatalf("ResolveMCPConfig succeeded with ambiguous top-level MCP server blocks")
	}
	if !strings.Contains(err.Error(), "ambiguous mcp server block fields") {
		t.Fatalf("error = %q, want ambiguous top-level block evidence", err.Error())
	}
	if len(resolved.Servers) != 0 {
		t.Fatalf("resolved valid servers = %#v, want none", resolved.Servers)
	}
	status, ok := resolved.Status("mcp_servers")
	if !ok {
		t.Fatalf("missing invalid top-level status in %#v", resolved.Statuses)
	}
	if status.Status != MCPConfigStatusInvalidConfig {
		t.Fatalf("status = %q, want %q (reason=%q)", status.Status, MCPConfigStatusInvalidConfig, status.Reason)
	}
}

func TestResolveMCPConfigEnabledServerStillRequiresTransport(t *testing.T) {
	resolved, err := ResolveMCPConfig(map[string]any{
		"mcp_servers": map[string]any{
			"broken": map[string]any{
				"enabled": true,
			},
		},
	}, MCPConfigOptions{LookupEnv: func(string) (string, bool) { return "", false }})
	if err == nil {
		t.Fatalf("ResolveMCPConfig succeeded for enabled server without transport")
	}

	status, ok := resolved.Status("broken")
	if !ok {
		t.Fatalf("missing invalid server status in %#v", resolved.Statuses)
	}
	if status.Status != MCPConfigStatusInvalidTransport {
		t.Fatalf("status = %q, want %q (reason=%q)", status.Status, MCPConfigStatusInvalidTransport, status.Reason)
	}
}

func TestResolveMCPConfigRejectsOverflowingTimeout(t *testing.T) {
	resolved, err := ResolveMCPConfig(map[string]any{
		"mcp_servers": map[string]any{
			"overflow": map[string]any{
				"command": "npx",
				"timeout": 1e300,
			},
		},
	}, MCPConfigOptions{LookupEnv: func(string) (string, bool) { return "", false }})
	if err == nil {
		t.Fatalf("ResolveMCPConfig succeeded for overflowing timeout")
	}
	if !strings.Contains(err.Error(), "timeout is too large") {
		t.Fatalf("error = %q, want timeout overflow evidence", err.Error())
	}
	if len(resolved.Servers) != 0 {
		t.Fatalf("resolved valid servers = %#v, want none", resolved.Servers)
	}

	status, ok := resolved.Status("overflow")
	if !ok {
		t.Fatalf("missing invalid server status in %#v", resolved.Statuses)
	}
	if status.Status != MCPConfigStatusInvalidConfig {
		t.Fatalf("status = %q, want %q (reason=%q)", status.Status, MCPConfigStatusInvalidConfig, status.Reason)
	}
}

func TestResolveMCPConfigRejectsOverflowingSamplingInteger(t *testing.T) {
	resolved, err := ResolveMCPConfig(map[string]any{
		"mcp_servers": map[string]any{
			"overflow": map[string]any{
				"command": "npx",
				"sampling": map[string]any{
					"max_tokens_cap": 1e300,
				},
			},
		},
	}, MCPConfigOptions{LookupEnv: func(string) (string, bool) { return "", false }})
	if err == nil {
		t.Fatalf("ResolveMCPConfig succeeded for overflowing sampling integer")
	}
	if !strings.Contains(err.Error(), "sampling.max_tokens_cap is too large") {
		t.Fatalf("error = %q, want sampling integer overflow evidence", err.Error())
	}
	if len(resolved.Servers) != 0 {
		t.Fatalf("resolved valid servers = %#v, want none", resolved.Servers)
	}

	status, ok := resolved.Status("overflow")
	if !ok {
		t.Fatalf("missing invalid server status in %#v", resolved.Statuses)
	}
	if status.Status != MCPConfigStatusInvalidConfig {
		t.Fatalf("status = %q, want %q (reason=%q)", status.Status, MCPConfigStatusInvalidConfig, status.Reason)
	}
}

func TestResolveMCPConfigRejectsAmbiguousCaseFoldedServerFields(t *testing.T) {
	tests := []struct {
		name   string
		server map[string]any
	}{
		{
			name: "case-only variants",
			server: map[string]any{
				"Command": "npx",
				"COMMAND": "uvx",
			},
		},
		{
			name: "exact plus folded variant",
			server: map[string]any{
				"command": "npx",
				"COMMAND": "uvx",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resolved, err := ResolveMCPConfig(map[string]any{
				"mcp_servers": map[string]any{
					"ambiguous": tc.server,
				},
			}, MCPConfigOptions{LookupEnv: func(string) (string, bool) { return "", false }})
			if err == nil {
				t.Fatalf("ResolveMCPConfig succeeded for ambiguous case-folded command fields")
			}
			if !strings.Contains(err.Error(), "ambiguous command field variants") {
				t.Fatalf("error = %q, want ambiguous command evidence", err.Error())
			}
			if strings.Contains(err.Error(), "npx") || strings.Contains(err.Error(), "uvx") {
				t.Fatalf("error leaked ambiguous command values: %s", err.Error())
			}
			if len(resolved.Servers) != 0 {
				t.Fatalf("resolved valid servers = %#v, want none", resolved.Servers)
			}

			status, ok := resolved.Status("ambiguous")
			if !ok {
				t.Fatalf("missing invalid server status in %#v", resolved.Statuses)
			}
			if status.Status != MCPConfigStatusInvalidConfig {
				t.Fatalf("status = %q, want %q (reason=%q)", status.Status, MCPConfigStatusInvalidConfig, status.Reason)
			}
		})
	}
}

func TestResolveMCPConfigRejectsAmbiguousCaseFoldedSamplingFields(t *testing.T) {
	resolved, err := ResolveMCPConfig(map[string]any{
		"mcp_servers": map[string]any{
			"ambiguous": map[string]any{
				"command": "npx",
				"sampling": map[string]any{
					"Max_RPM": "8",
					"MAX_RPM": "9",
				},
			},
		},
	}, MCPConfigOptions{LookupEnv: func(string) (string, bool) { return "", false }})
	if err == nil {
		t.Fatalf("ResolveMCPConfig succeeded for ambiguous case-folded sampling fields")
	}
	if !strings.Contains(err.Error(), "ambiguous sampling.max_rpm field variants") {
		t.Fatalf("error = %q, want ambiguous sampling field evidence", err.Error())
	}
	if strings.Contains(err.Error(), "8") || strings.Contains(err.Error(), "9") {
		t.Fatalf("error leaked ambiguous sampling values: %s", err.Error())
	}
	if len(resolved.Servers) != 0 {
		t.Fatalf("resolved valid servers = %#v, want none", resolved.Servers)
	}

	status, ok := resolved.Status("ambiguous")
	if !ok {
		t.Fatalf("missing invalid server status in %#v", resolved.Statuses)
	}
	if status.Status != MCPConfigStatusInvalidConfig {
		t.Fatalf("status = %q, want %q (reason=%q)", status.Status, MCPConfigStatusInvalidConfig, status.Reason)
	}
}

func TestResolveMCPConfigRedactsHTTPURLCredentialsInStatus(t *testing.T) {
	resolved, err := ResolveMCPConfig(map[string]any{
		"mcp_servers": map[string]any{
			"remote": map[string]any{
				"url": "https://mcp.example.test/sse?token=super-secret-token&safe=1",
			},
		},
	}, MCPConfigOptions{LookupEnv: func(string) (string, bool) { return "", false }})
	if err != nil {
		t.Fatalf("ResolveMCPConfig returned error: %v", err)
	}

	status, ok := resolved.Status("remote")
	if !ok {
		t.Fatalf("missing remote status in %#v", resolved.Statuses)
	}
	if strings.Contains(status.URL, "super-secret-token") {
		t.Fatalf("status URL leaked credential: %q", status.URL)
	}
	if !strings.Contains(status.URL, RedactedMCPConfigValue) {
		t.Fatalf("status URL = %q, want redaction marker", status.URL)
	}

	text := resolved.RedactedStatusText()
	if strings.Contains(text, "super-secret-token") {
		t.Fatalf("status text leaked credential: %s", text)
	}
	if !strings.Contains(text, "url=https://mcp.example.test/sse?"+RedactedMCPConfigValue+"&safe=1") {
		t.Fatalf("status text = %q, want redacted URL credential", text)
	}
}

func TestResolveMCPTransportClassifiesExactlyOneTransport(t *testing.T) {
	tests := []struct {
		name          string
		command       string
		url           string
		wantTransport MCPTransport
		wantError     string
	}{
		{name: "stdio command", command: "npx", wantTransport: MCPTransportStdio},
		{name: "http url", url: "https://mcp.example.test/mcp", wantTransport: MCPTransportHTTP},
		{name: "both command and url", command: "npx", url: "https://mcp.example.test/mcp", wantError: "both command and url"},
		{name: "neither command nor url", wantError: "requires command or url"},
		{name: "invalid url", url: "ftp://mcp.example.test/mcp", wantError: "url scheme must be http or https"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveMCPTransport(tc.command, tc.url)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("resolveMCPTransport error = %v, want %q", err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveMCPTransport returned error: %v", err)
			}
			if got != tc.wantTransport {
				t.Fatalf("transport = %q, want %q", got, tc.wantTransport)
			}
		})
	}
}

func TestResolveMCPConfigRejectsInvalidHTTPURLs(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{name: "unsupported scheme", url: "ftp://mcp.example.test/mcp", want: "url scheme must be http or https"},
		{name: "missing host", url: "https:///mcp", want: "url must include a host"},
		{name: "malformed", url: "://mcp.example.test", want: "url is invalid"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resolved, err := ResolveMCPConfig(map[string]any{
				"mcp_servers": map[string]any{
					"remote": map[string]any{
						"url": tc.url,
					},
				},
			}, MCPConfigOptions{LookupEnv: func(string) (string, bool) { return "", false }})
			if err == nil {
				t.Fatalf("ResolveMCPConfig succeeded for invalid HTTP URL %q", tc.url)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %q, want %q", err.Error(), tc.want)
			}
			if strings.Contains(err.Error(), tc.url) {
				t.Fatalf("error leaked raw invalid URL: %s", err.Error())
			}
			if len(resolved.Servers) != 0 {
				t.Fatalf("resolved valid servers = %#v, want none", resolved.Servers)
			}

			status, ok := resolved.Status("remote")
			if !ok {
				t.Fatalf("missing invalid server status in %#v", resolved.Statuses)
			}
			if status.Status != MCPConfigStatusInvalidTransport {
				t.Fatalf("status = %q, want %q (reason=%q)", status.Status, MCPConfigStatusInvalidTransport, status.Reason)
			}
		})
	}
}

func TestResolveMCPConfigRejectsEmptyEnvInterpolationReference(t *testing.T) {
	resolved, err := ResolveMCPConfig(map[string]any{
		"mcp_servers": map[string]any{
			"broken": map[string]any{
				"command": "npx",
				"env": map[string]any{
					"API_KEY": "${}",
				},
			},
		},
	}, MCPConfigOptions{LookupEnv: func(string) (string, bool) { return "", false }})
	if err == nil {
		t.Fatalf("ResolveMCPConfig succeeded for empty env interpolation reference")
	}
	if !strings.Contains(err.Error(), "invalid env variable name") {
		t.Fatalf("error = %q, want invalid env variable evidence", err.Error())
	}
	if len(resolved.Servers) != 0 {
		t.Fatalf("resolved valid servers = %#v, want none", resolved.Servers)
	}

	status, ok := resolved.Status("broken")
	if !ok {
		t.Fatalf("missing invalid server status in %#v", resolved.Statuses)
	}
	if status.Status != MCPConfigStatusInvalidEnv {
		t.Fatalf("status = %q, want %q (reason=%q)", status.Status, MCPConfigStatusInvalidEnv, status.Reason)
	}
}

func TestResolveMCPConfigDisabledServerDoesNotRequireTransport(t *testing.T) {
	resolved, err := ResolveMCPConfig(map[string]any{
		"mcp_servers": map[string]any{
			"parked": map[string]any{
				"enabled": false,
			},
		},
	}, MCPConfigOptions{LookupEnv: func(string) (string, bool) { return "", false }})
	if err != nil {
		t.Fatalf("ResolveMCPConfig returned error for disabled server without transport: %v", err)
	}

	status, ok := resolved.Status("parked")
	if !ok {
		t.Fatalf("missing disabled server status in %#v", resolved.Statuses)
	}
	if status.Status != MCPConfigStatusDisabled {
		t.Fatalf("status = %q, want %q (reason=%q)", status.Status, MCPConfigStatusDisabled, status.Reason)
	}
	if status.Enabled {
		t.Fatalf("status.Enabled = true, want false")
	}

	server, ok := resolved.Server("parked")
	if !ok {
		t.Fatalf("missing disabled server definition in %#v", resolved.Servers)
	}
	if server.Enabled {
		t.Fatalf("server.Enabled = true, want false")
	}
}
