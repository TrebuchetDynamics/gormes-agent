package slack

import (
	"os"

	"github.com/spf13/cobra"

	appslack "github.com/TrebuchetDynamics/gormes-agent/internal/app/slack"
	channelsmodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/channels"
)

func NewCommand() *cobra.Command {
	return channelsmodule.NewSlackCommandWithSeams(channelsmodule.SlackCommandSeams{
		Manifest: RunManifestCommand,
	})
}

func RunManifestCommand(cmd *cobra.Command, opts channelsmodule.SlackManifestOptions) error {
	return appslack.RunManifest(cmd.OutOrStdout(), cmd.ErrOrStderr(), appslack.ManifestOptions{
		BotName:      opts.BotName,
		Description:  opts.Description,
		SlashesOnly:  opts.SlashesOnly,
		WriteChanged: opts.WriteChanged,
		WriteTarget:  opts.WriteTarget,
	})
}

func ManifestPayload(botName, description string, slashesOnly bool) (any, error) {
	return appslack.ManifestPayload(botName, description, slashesOnly)
}

func ManifestSlashCommands(requestURL string) []map[string]any {
	return appslack.ManifestSlashCommands(requestURL)
}

func SanitizeManifestName(name string) string { return appslack.SanitizeManifestName(name) }

func ClampString(s string, max int) string { return appslack.ClampString(s, max) }

func NonEmpty(value, fallback string) string { return appslack.NonEmpty(value, fallback) }

func ExpandUserPath(path string) string { return appslack.ExpandUserPath(path) }

func WriteFileAtomic(path string, body []byte, perm os.FileMode) error {
	return appslack.WriteFileAtomic(path, body, perm)
}
