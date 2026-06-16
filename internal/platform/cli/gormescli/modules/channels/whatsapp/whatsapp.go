package whatsapp

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/channels/internal/seam"
)

type WhatsAppOptions struct {
	Mode         string
	AllowedUsers string
	AllowAll     bool
	Debug        bool
	PlanOnly     bool
	JSONOut      bool
	BridgeScript string
}

type WhatsAppCommandSeams struct {
	Run func(*cobra.Command, WhatsAppOptions) error
}

func NewWhatsAppCommandWithSeams(seams WhatsAppCommandSeams) *cobra.Command {
	seams = seams.withDefaults()
	opts := WhatsAppOptions{}
	cmd := &cobra.Command{
		Use:          "whatsapp",
		Short:        "Set up WhatsApp pairing through the Hermes-compatible Baileys bridge",
		Long:         "Sets up WhatsApp mode, allowlist state, bridge dependencies, and QR pairing through the Hermes-compatible Baileys bridge.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return seams.Run(cmd, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Mode, "mode", "bot", "WhatsApp mode: bot or self-chat")
	cmd.Flags().StringVar(&opts.AllowedUsers, "allowed-users", "", "comma-separated allowed phone numbers with country code and no punctuation")
	cmd.Flags().BoolVar(&opts.AllowAll, "allow-all-users", false, "render allow-all sender configuration")
	cmd.Flags().BoolVar(&opts.Debug, "debug", false, "render WHATSAPP_DEBUG=true in the dotenv plan")
	cmd.Flags().BoolVar(&opts.PlanOnly, "plan", false, "render the WhatsApp bridge plan without starting QR pairing")
	cmd.Flags().BoolVar(&opts.JSONOut, "json", false, "with --plan, emit the plan as machine-readable JSON")
	cmd.Flags().StringVar(&opts.BridgeScript, "bridge-script", "", "override the WhatsApp bridge.js path")
	return cmd
}

func (s WhatsAppCommandSeams) withDefaults() WhatsAppCommandSeams {
	if s.Run == nil {
		s.Run = func(*cobra.Command, WhatsAppOptions) error {
			return seam.Missing("whatsapp run")
		}
	}
	return s
}
