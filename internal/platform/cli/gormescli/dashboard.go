package gormescli

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/dashboardruntime"
)

type DashboardCommandOptions = dashboardruntime.DashboardCommandOptions

func NewDashboardCommand(options DashboardCommandOptions) *cobra.Command {
	return dashboardruntime.NewDashboardCommand(options)
}
