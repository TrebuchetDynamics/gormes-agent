package main

import (
	"github.com/spf13/cobra"

	channelsmodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/channels"
)

type whatsappCommandSeams = channelsmodule.WhatsAppAppSeams
type whatsappPairingPlan = channelsmodule.WhatsAppPairingPlan

func newWhatsAppCommand() *cobra.Command {
	return channelsmodule.NewWhatsAppAppCommand(whatsappCommandOptions())
}

func newWhatsAppCommandWithSeams(seams whatsappCommandSeams) *cobra.Command {
	return channelsmodule.NewWhatsAppAppCommandWithSeams(seams, whatsappCommandOptions())
}

func whatsappCommandOptions() channelsmodule.WhatsAppAppOptions {
	return channelsmodule.WhatsAppAppOptions{BuildProvenance: func() channelsmodule.WhatsAppBuildProvenance {
		build := newBuildProvenance()
		return channelsmodule.WhatsAppBuildProvenance{
			Version:   build.Version,
			GitCommit: build.GitCommit,
		}
	}}
}
