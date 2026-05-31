package config

import (
	"strings"
	"testing"
)

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
	resolved, err := ResolveMCPConfig(map[string]any{
		"mcp_servers": map[string]any{
			"ambiguous": map[string]any{
				"Command": "npx",
				"COMMAND": "uvx",
			},
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
