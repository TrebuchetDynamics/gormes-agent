package gormescli

import (
	"github.com/spf13/cobra"

	appplugins "github.com/TrebuchetDynamics/gormes-agent/internal/app/plugins"
	pluginspkg "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/plugins"
)

// NewPluginsCommand builds the plugins command tree through the shared CLI facade.
func NewPluginsCommand(build func() BuildProvenance) *cobra.Command {
	return appplugins.NewCommand(pluginsOptions(build))
}

// NewPluginsCommandWithManager injects a lifecycle manager for CLI contract tests.
func NewPluginsCommandWithManager(manager *pluginspkg.LifecycleManager, build func() BuildProvenance) *cobra.Command {
	return appplugins.NewCommandWithManager(manager, pluginsOptions(build))
}

func pluginsOptions(build func() BuildProvenance) appplugins.Options {
	return appplugins.Options{
		BuildProvenance: func() appplugins.BuildProvenance {
			if build == nil {
				return appplugins.BuildProvenance{}
			}
			provenance := build()
			return appplugins.BuildProvenance{Version: provenance.Version, GitCommit: provenance.GitCommit}
		},
	}
}
