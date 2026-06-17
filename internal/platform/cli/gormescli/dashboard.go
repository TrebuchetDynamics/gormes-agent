package gormescli

import (
	"github.com/spf13/cobra"

	appdashboard "github.com/TrebuchetDynamics/gormes-agent/internal/app/dashboard"
)

type DashboardCommandOptions = appdashboard.CommandOptions

func NewDashboardCommand(options DashboardCommandOptions) *cobra.Command {
	return appdashboard.NewCommand(options)
}

func DefaultDashboardCommandOptions(version string, gitCommit string, gitDirty bool) DashboardCommandOptions {
	opts := appdashboard.DefaultCommandOptions(version, gitCommit, gitDirty)
	opts.BuildTurnLoop = buildDashboardTurnLoop
	return opts
}
