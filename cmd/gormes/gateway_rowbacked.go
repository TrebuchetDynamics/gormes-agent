package main

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/app/gormescli"
	gatewaymodule "github.com/TrebuchetDynamics/gormes-agent/internal/app/gormescli/modules/gateway"
)

func newWebhookCommand() *cobra.Command {
	return gatewaymodule.NewWebhookCommand(gatewayCommandOptions())
}

func newHooksCommand() *cobra.Command {
	return gatewaymodule.NewHooksCommand(gatewayCommandOptions())
}

func newPairingCommand() *cobra.Command {
	return gatewaymodule.NewPairingCommand(gatewayCommandOptions())
}

func gatewayCommandOptions() gatewaymodule.Options {
	return gatewaymodule.Options{
		BuildProvenance: func() gormescli.BuildProvenance {
			build := newBuildProvenance()
			return gormescli.BuildProvenance{
				Version:   build.Version,
				GitCommit: build.GitCommit,
			}
		},
	}
}
