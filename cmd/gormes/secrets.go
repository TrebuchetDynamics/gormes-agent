package main

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

func newSecretsCommand() *cobra.Command {
	return gormescli.NewSecretsCommand(func() gormescli.BuildProvenance {
		build := newBuildProvenance()
		return gormescli.BuildProvenance{
			Version:   build.Version,
			GitCommit: build.GitCommit,
		}
	})
}
