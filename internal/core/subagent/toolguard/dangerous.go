// internal/core/subagent/toolguard/dangerous.go
package toolguard

import (
	"context"
	"encoding/json"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// GuardDangerousCommand applies the standard dangerous-command approval policy
// for child tool executions. It is pure except for the approval callback carried
// by ctx through the tools package.
func GuardDangerousCommand(ctx context.Context, approvalMode string, req tools.ToolRequest) tools.BlockedResult {
	cmd := ChildCommand(req)
	if cmd == "" {
		return tools.BlockedResult{}
	}
	result := tools.GuardCommandWithApproval(ctx, req.ToolName, cmd, approvalMode)
	if result.Description == "" || result.Approved {
		return tools.BlockedResult{}
	}
	return result
}

// ChildCommand extracts the shell/code command text from tool requests that can
// execute process-like work in a child agent.
func ChildCommand(req tools.ToolRequest) string {
	switch req.ToolName {
	case "terminal", "execute_code":
	default:
		return ""
	}
	var payload struct {
		Command string `json:"command"`
		Code    string `json:"code"`
	}
	if err := json.Unmarshal(req.Input, &payload); err != nil {
		return ""
	}
	if payload.Command != "" {
		return payload.Command
	}
	return payload.Code
}
