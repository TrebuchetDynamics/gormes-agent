package main

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

type routerCommandOptions = gormescli.RouterOptions

func newRouterCommand() *cobra.Command {
	return newRouterCommandWithOptions(routerCommandOptions{})
}

func newRouterCommandWithOptions(opts routerCommandOptions) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:          "router",
		Short:        "Inspect or run the local OpenAI-compatible Gormes Router",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return gormescli.RunRouter(cmd.Context(), cmd.OutOrStdout(), gormescli.RouterRequest{DryRun: dryRun}, opts)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "render Router listen/status/client config without binding a port")
	return cmd
}
