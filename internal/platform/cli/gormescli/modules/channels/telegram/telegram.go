package telegram

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/channels/internal/seam"
)

type TelegramCommandSeams struct {
	Run func(*cobra.Command, []string) error
}

func NewTelegramCommandWithSeams(seams TelegramCommandSeams) *cobra.Command {
	seams = seams.withDefaults()
	return &cobra.Command{
		Use:          "telegram",
		Short:        "Run Gormes as a Telegram bot adapter",
		Long:         "Long-polls Telegram for DMs from the allowlisted chat, drives the same kernel + tool loop as the TUI, and persists turns to the SQLite memory store.",
		SilenceUsage: true,
		RunE:         seams.Run,
	}
}

func (s TelegramCommandSeams) withDefaults() TelegramCommandSeams {
	if s.Run == nil {
		s.Run = func(*cobra.Command, []string) error {
			return seam.Missing("telegram run")
		}
	}
	return s
}
