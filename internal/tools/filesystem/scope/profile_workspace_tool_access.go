package scope

import "fmt"

const ProfileWorkspaceToolAccessExecuteBlocked = "profile_workspace_tool_access_execute_blocked"

type ProfileWorkspaceToolAccessDenial struct {
	Evidence string
	Reason   string
	Message  string
}

func ProfileWorkspaceExecuteDenied(toolName string) ProfileWorkspaceToolAccessDenial {
	if toolName == "" {
		toolName = "tool"
	}
	return ProfileWorkspaceToolAccessDenial{
		Evidence: ProfileWorkspaceScopeViolation,
		Reason:   ProfileWorkspaceToolAccessExecuteBlocked,
		Message:  fmt.Sprintf("%s: %s: %s cannot prove confinement for a non-empty profile workspace allow-list; fail closed before spawning", ProfileWorkspaceScopeViolation, ProfileWorkspaceToolAccessExecuteBlocked, toolName),
	}
}
