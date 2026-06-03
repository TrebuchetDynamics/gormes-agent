package main

import (
	"github.com/spf13/cobra"

	pluginspkg "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/plugins"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

func newPluginsCommand() *cobra.Command {
	return gormescli.NewPluginsCommand(pluginBuildProvenance)
}

func newPluginsCommandWithManager(manager *pluginspkg.LifecycleManager) *cobra.Command {
	return gormescli.NewPluginsCommandWithManager(manager, pluginBuildProvenance)
}

func pluginBuildProvenance() gormescli.BuildProvenance {
	build := newBuildProvenance()
	return gormescli.BuildProvenance{
		Version:   build.Version,
		GitCommit: build.GitCommit,
	}
}
