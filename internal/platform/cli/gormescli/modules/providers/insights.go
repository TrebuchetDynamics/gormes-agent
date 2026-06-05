package providers

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

// NewInsightsCommand creates the provider-owned insights command surface. The
// runtime telemetry implementation is still row-backed; the command surface is
// owned here so provider command migration does not depend on cmd/gormes.
func NewInsightsCommand(opts Options) *cobra.Command {
	return gormescli.NewRowBackedCommand(gormescli.RowBackedCommandSpec{
		Use:   "insights",
		Short: "Show Hermes-compatible runtime insights",
		Row:   "Self-monitoring telemetry",
	}, gormescli.RowBackedCommandOptions{
		BuildProvenance: opts.BuildProvenance,
	})
}
