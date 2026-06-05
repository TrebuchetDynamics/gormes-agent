package gormescli

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	approuter "github.com/TrebuchetDynamics/gormes-agent/internal/app/router"
)

type RouterOptions = approuter.Options
type RouterRequest = approuter.Request
type RouterReadModel = approuter.ReadModel

func NewRouterCommand() *cobra.Command {
	return NewRouterCommandWithOptions(RouterOptions{})
}

func NewRouterCommandWithOptions(opts RouterOptions) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:          "router",
		Short:        "Inspect or run the local OpenAI-compatible Gormes Router",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return RunRouter(cmd.Context(), cmd.OutOrStdout(), RouterRequest{DryRun: dryRun}, opts)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "render Router listen/status/client config without binding a port")
	return cmd
}

func RunRouter(ctx context.Context, out io.Writer, request RouterRequest, opts RouterOptions) error {
	return approuter.Run(ctx, out, request, opts)
}

func PrintRouterDryRun(out io.Writer, model RouterReadModel) {
	approuter.PrintDryRun(out, model)
}

func RouterOpenAIBaseURL(listen string) string {
	return approuter.OpenAIBaseURL(listen)
}
