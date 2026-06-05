package gateway

import (
	"github.com/spf13/cobra"

	rowbackedcmd "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/gateway/rowbacked"
)

const GatewayCronRow = rowbackedcmd.GatewayCronRow

type Options = rowbackedcmd.Options

func NewWebhookCommand(opts Options) *cobra.Command {
	return rowbackedcmd.NewWebhookCommand(opts)
}

func NewHooksCommand(opts Options) *cobra.Command {
	return rowbackedcmd.NewHooksCommand(opts)
}

func NewPairingCommand(opts Options) *cobra.Command {
	return rowbackedcmd.NewPairingCommand(opts)
}
