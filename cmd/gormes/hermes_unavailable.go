package main

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

type hermesUnavailableCommandSpec = gormescli.RowBackedCommandSpec

func newHermesUnavailableCommand(spec hermesUnavailableCommandSpec, children ...*cobra.Command) *cobra.Command {
	return gormescli.NewRowBackedCommand(spec, hermesUnavailableOptions(), children...)
}

func newHermesUnavailableParent(use, short string, children ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
	}
	cmd.AddCommand(children...)
	return cmd
}

func hermesUnavailableOptions() gormescli.RowBackedCommandOptions {
	return gormescli.RowBackedCommandOptions{BuildProvenance: func() gormescli.BuildProvenance {
		build := newBuildProvenance()
		return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
	}}
}

func hermesUnavailableYesFlag(cmd *cobra.Command) {
	cmd.Flags().BoolP("yes", "y", false, "skip confirmation")
}
