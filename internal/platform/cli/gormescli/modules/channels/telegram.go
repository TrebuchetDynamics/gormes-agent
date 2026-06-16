package channels

import (
	"github.com/spf13/cobra"

	telegramcmd "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/channels/telegram"
)

type TelegramCommandSeams = telegramcmd.TelegramCommandSeams

func NewTelegramCommandWithSeams(seams TelegramCommandSeams) *cobra.Command {
	return telegramcmd.NewTelegramCommandWithSeams(seams)
}
