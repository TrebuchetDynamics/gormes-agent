package toolguard

import (
	"encoding/json"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestChildCommandExtractsExecutablePayloads(t *testing.T) {
	tests := []struct {
		name string
		req  tools.ToolRequest
		want string
	}{
		{
			name: "terminal command",
			req:  tools.ToolRequest{ToolName: "terminal", Input: mustJSON(t, map[string]string{"command": "rm -rf /tmp/example"})},
			want: "rm -rf /tmp/example",
		},
		{
			name: "execute code fallback",
			req:  tools.ToolRequest{ToolName: "execute_code", Input: mustJSON(t, map[string]string{"code": "print('hello')"})},
			want: "print('hello')",
		},
		{
			name: "non executable tool",
			req:  tools.ToolRequest{ToolName: "memory", Input: mustJSON(t, map[string]string{"command": "ignored"})},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ChildCommand(tt.req); got != tt.want {
				t.Fatalf("ChildCommand() = %q, want %q", got, tt.want)
			}
		})
	}
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
