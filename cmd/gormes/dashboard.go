package main

import (
	"github.com/spf13/cobra"

	dashboardcmd "github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/dashboard"
)

func newDashboardCommand() *cobra.Command {
	return dashboardcmd.NewCommand(dashboardcmd.DefaultCommandOptions(Version, resolveGitCommit(), resolveGitDirty()))
}
