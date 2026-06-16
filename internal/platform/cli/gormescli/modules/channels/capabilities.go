package channels

import (
	"github.com/spf13/cobra"

	channelcaps "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels"
	channelcapscmd "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/channels/capabilities"
)

type Options = channelcapscmd.Options
type Seams = channelcapscmd.Seams

func DefaultSeams() Seams {
	return channelcapscmd.DefaultSeams()
}

func NewCommand(opts Options) *cobra.Command {
	return channelcapscmd.NewCommand(opts)
}

func NewCommandWithSeams(seams Seams, opts Options) *cobra.Command {
	return channelcapscmd.NewCommandWithSeams(seams, opts)
}

func RenderCapabilitiesText(reports []channelcaps.CapabilityReport) string {
	return channelcapscmd.RenderCapabilitiesText(reports)
}

func isQuickSetupChannel(channel string) bool {
	return channelcapscmd.IsQuickSetupChannel(channel)
}
