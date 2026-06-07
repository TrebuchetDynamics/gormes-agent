package channels

import (
	"github.com/spf13/cobra"

	slackcmd "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/channels/slack"
)

const SlackManifestDefaultWrite = slackcmd.SlackManifestDefaultWrite

type SlackManifestOptions = slackcmd.SlackManifestOptions
type SlackCommandSeams = slackcmd.SlackCommandSeams

func NewSlackCommandWithSeams(seams SlackCommandSeams) *cobra.Command {
	return slackcmd.NewSlackCommandWithSeams(seams)
}
