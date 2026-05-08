package main

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// TestMCPLoginCommand_TextModeDoesNotDuplicateError pins the
// regression where `gormes mcp login <missing-server>` emitted the
// same error twice — once on stdout via fmt.Fprintln(result.Error())
// AND once on stderr as cobra's standard `Error: ...` rendering of
// the returned error. Operators saw:
//
//   evidence=mcp_server_unknown server=missing unknown MCP server
//   Error: evidence=mcp_server_unknown server=missing unknown MCP server
//
// Two lines, same content. The fix is to suppress the redundant
// stdout print on the error path; cobra's stderr rendering carries
// the message, and JSON mode is unaffected. The contract: the
// evidence string must appear EXACTLY ONCE across stdout+stderr.
func TestMCPLoginCommand_TextModeDoesNotDuplicateError(t *testing.T) {
	cmd := newMCPCommandWithRuntime(mcpLoginRuntime{
		loadConfig: func() (tools.MCPConfigResolution, error) {
			return commandMCPResolution(), nil
		},
		store: tools.NewMCPOAuthStore(),
	})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "login", "missing")
	if err == nil {
		t.Fatalf("missing server must error; stdout=%s stderr=%s", stdout, stderr)
	}
	combined := stdout + "\n" + stderr
	count := strings.Count(combined, "mcp_server_unknown")
	if count != 1 {
		t.Fatalf("`mcp_server_unknown` must appear EXACTLY once across stdout+stderr (was %d times):\nstdout=%s\nstderr=%s", count, stdout, stderr)
	}
}
