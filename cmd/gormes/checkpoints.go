package main

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

func newCheckpointsCommand() *cobra.Command {
	return gormescli.NewCheckpointsCommand(func() gormescli.BuildProvenance {
		provenance := newBuildProvenance()
		return gormescli.BuildProvenance{Version: provenance.Version, GitCommit: provenance.GitCommit}
	})
}
