package main

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

func newPluginsCommand() *cobra.Command {
	return gormescli.NewPluginsCommand(pluginCommandOptions())
}

func newPluginsCommandWithManager(manager any) *cobra.Command {
	return gormescli.NewPluginsCommandWithManager(manager, pluginCommandOptions())
}

func pluginCommandOptions() gormescli.PluginsOptions {
	return gormescli.PluginsOptions{BuildProvenance: pluginBuildProvenance}
}

func pluginBuildProvenance() gormescli.PluginsBuildProvenance {
	build := newBuildProvenance()
	return gormescli.PluginsBuildProvenance{
		Version:   build.Version,
		GitCommit: build.GitCommit,
	}
}
