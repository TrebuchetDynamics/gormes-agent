package main

import (
	"github.com/spf13/cobra"

	whatsappcmd "github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/whatsapp"
)

type whatsappCommandSeams = whatsappcmd.Seams
type whatsappPairingPlan = whatsappcmd.PairingPlan

func newWhatsAppCommand() *cobra.Command {
	return whatsappcmd.NewCommand(whatsappCommandOptions())
}

func newWhatsAppCommandWithSeams(seams whatsappCommandSeams) *cobra.Command {
	return whatsappcmd.NewCommandWithSeams(seams, whatsappCommandOptions())
}

func whatsappCommandOptions() whatsappcmd.Options {
	return whatsappcmd.Options{BuildProvenance: func() whatsappcmd.BuildProvenance {
		build := newBuildProvenance()
		return whatsappcmd.BuildProvenance{
			Version:   build.Version,
			GitCommit: build.GitCommit,
		}
	}}
}
