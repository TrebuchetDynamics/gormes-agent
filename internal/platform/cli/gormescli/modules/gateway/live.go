package gateway

import (
	"github.com/spf13/cobra"

	livecmd "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/gateway/live"
)

var MutatingUnavailableSubcommands = livecmd.MutatingUnavailableSubcommands
var RowBackedUnavailableSubcommands = livecmd.RowBackedUnavailableSubcommands

type GatewayCommandSeams = livecmd.GatewayCommandSeams

func NewGatewayCommandWithSeams(seams GatewayCommandSeams, opts Options) *cobra.Command {
	return livecmd.NewGatewayCommandWithSeams(seams, opts)
}
