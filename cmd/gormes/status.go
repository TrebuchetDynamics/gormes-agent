package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func newStatusCommand() *cobra.Command {
	return gormescli.NewStatusCommand(gormescli.StatusCommandOptions{
		BuildProvenance: statusBuildProvenance,
		SystemSnapshot:  statusSystemSnapshot,
	})
}

func statusBuildProvenance() gormescli.BuildProvenance {
	build := newBuildProvenance()
	return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
}

func statusSystemSnapshot(ctx context.Context) (toolspkg.SystemEventsSnapshot, error) {
	return cliSystemEventsManager().Snapshot(ctx)
}
