package routercmd

import (
	"io"

	"github.com/spf13/cobra"

	approuter "github.com/TrebuchetDynamics/gormes-agent/internal/app/router"
)

type Options = approuter.Options
type ReadModel = approuter.ReadModel

func New(opts Options) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:          "router",
		Short:        "Inspect or run the local OpenAI-compatible Gormes Router",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return approuter.Run(cmd.Context(), cmd.OutOrStdout(), approuter.Request{DryRun: dryRun}, opts)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "render Router listen/status/client config without binding a port")
	return cmd
}

func PrintDryRun(out io.Writer, model ReadModel) {
	approuter.PrintDryRun(out, model)
}

func OpenAIBaseURL(listen string) string {
	return approuter.OpenAIBaseURL(listen)
}
