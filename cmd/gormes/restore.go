package main

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

type restoreCommandSeams = gormescli.RestoreCommandSeams

func newRestoreCommand() *cobra.Command {
	return gormescli.NewRestoreCommand(restoreBuildProvenance, emitJSONInputError)
}

func newRestoreCommandWithSeams(seams restoreCommandSeams) *cobra.Command {
	return gormescli.NewRestoreCommandWithSeams(seams, restoreBuildProvenance, emitJSONInputError)
}

func restoreBuildProvenance() gormescli.BuildProvenance {
	build := newBuildProvenance()
	return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
}
