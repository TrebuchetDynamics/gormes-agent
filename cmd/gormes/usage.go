package main

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	providermodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/providers"
)

type usageInvocation = providermodule.UsageInvocation

func newUsageCommand() *cobra.Command {
	return providermodule.NewUsageCommand(providerCommandOptions())
}

func runUsageCommand(cmd *cobra.Command, invocation usageInvocation) error {
	return providermodule.RunUsageCommand(cmd, invocation, providerCommandOptions())
}

func providerCommandOptions() providermodule.Options {
	return providermodule.Options{
		BuildProvenance: func() gormescli.BuildProvenance {
			build := newBuildProvenance()
			return gormescli.BuildProvenance{
				Version:   build.Version,
				GitCommit: build.GitCommit,
			}
		},
	}
}
