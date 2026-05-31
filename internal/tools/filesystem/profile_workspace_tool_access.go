package filesystem

import filescope "github.com/TrebuchetDynamics/gormes-agent/internal/tools/filesystem/scope"

const ProfileWorkspaceToolAccessExecuteBlocked = filescope.ProfileWorkspaceToolAccessExecuteBlocked

type ProfileWorkspaceToolAccessDenial = filescope.ProfileWorkspaceToolAccessDenial

func ProfileWorkspaceExecuteDenied(toolName string) ProfileWorkspaceToolAccessDenial {
	return filescope.ProfileWorkspaceExecuteDenied(toolName)
}
