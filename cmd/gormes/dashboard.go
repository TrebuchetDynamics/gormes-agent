package main

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

func newDashboardCommand() *cobra.Command {
	return gormescli.NewDashboardCommandForBinary(Version, resolveGitCommit(), resolveGitDirty())
}
