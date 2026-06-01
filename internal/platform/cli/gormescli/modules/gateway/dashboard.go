package gateway

import (
	"github.com/spf13/cobra"

	dashboardcmd "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/gateway/dashboard"
)

type DashboardOptions = dashboardcmd.DashboardOptions
type DashboardCommandSeams = dashboardcmd.DashboardCommandSeams

func NewDashboardCommandWithSeams(seams DashboardCommandSeams) *cobra.Command {
	return dashboardcmd.NewDashboardCommandWithSeams(seams)
}
