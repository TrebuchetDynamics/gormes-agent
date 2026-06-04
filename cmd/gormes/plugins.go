package main

import (
	"github.com/spf13/cobra"

	plugincmd "github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/plugins"
)

func newPluginsCommand() *cobra.Command {
	return plugincmd.NewCommand(pluginCommandOptions())
}

func newPluginsCommandWithManager(manager any) *cobra.Command {
	return plugincmd.NewCommandWithManager(manager, pluginCommandOptions())
}

func pluginCommandOptions() plugincmd.Options {
	return plugincmd.Options{BuildProvenance: pluginBuildProvenance}
}

func pluginBuildProvenance() plugincmd.BuildProvenance {
	build := newBuildProvenance()
	return plugincmd.BuildProvenance{
		Version:   build.Version,
		GitCommit: build.GitCommit,
	}
}
