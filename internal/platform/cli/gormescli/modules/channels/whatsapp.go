package channels

import (
	"github.com/spf13/cobra"

	whatsappcmd "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/channels/whatsapp"
)

type WhatsAppOptions = whatsappcmd.WhatsAppOptions
type WhatsAppCommandSeams = whatsappcmd.WhatsAppCommandSeams

func NewWhatsAppCommandWithSeams(seams WhatsAppCommandSeams) *cobra.Command {
	return whatsappcmd.NewWhatsAppCommandWithSeams(seams)
}
