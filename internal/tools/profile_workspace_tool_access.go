package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/filesystem"

const ProfileWorkspaceToolAccessExecuteBlocked = filesystem.ProfileWorkspaceToolAccessExecuteBlocked

type ProfileWorkspaceToolAccessDenial = filesystem.ProfileWorkspaceToolAccessDenial

func profileWorkspaceExecuteDenied(toolName string) ProfileWorkspaceToolAccessDenial {
	return filesystem.ProfileWorkspaceExecuteDenied(toolName)
}
