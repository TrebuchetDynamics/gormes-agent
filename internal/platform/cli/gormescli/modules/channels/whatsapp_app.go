package channels

import (
	"github.com/spf13/cobra"

	appwhatsapp "github.com/TrebuchetDynamics/gormes-agent/internal/app/whatsapp"
)

type WhatsAppBuildProvenance = appwhatsapp.BuildProvenance
type WhatsAppAppOptions = appwhatsapp.Options
type WhatsAppAppSeams = appwhatsapp.Seams
type WhatsAppPairingPlan = appwhatsapp.PairingPlan

func NewWhatsAppAppCommand(opts WhatsAppAppOptions) *cobra.Command {
	return appwhatsapp.NewCommand(opts)
}

func NewWhatsAppAppCommandWithSeams(seams WhatsAppAppSeams, opts WhatsAppAppOptions) *cobra.Command {
	return appwhatsapp.NewCommandWithSeams(seams, opts)
}
