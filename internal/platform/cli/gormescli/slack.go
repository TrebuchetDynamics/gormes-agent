package gormescli

import (
	"io/fs"

	"github.com/spf13/cobra"

	appslack "github.com/TrebuchetDynamics/gormes-agent/internal/app/slack"
)

type SlackManifestOptions struct {
	BotName      string
	Description  string
	SlashesOnly  bool
	WriteTarget  string
	WriteChanged bool
}

func NewSlackCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "slack",
		Short: "Slack integration helpers",
	}
	cmd.AddCommand(newSlackManifestCommand())
	return cmd
}

func newSlackManifestCommand() *cobra.Command {
	opts := SlackManifestOptions{}
	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "Print or write a Slack app manifest",
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts.WriteChanged = cmd.Flags().Changed("write")
			return RunSlackManifestCommand(cmd, opts)
		},
	}
	cmd.Flags().StringVar(&opts.WriteTarget, "write", "", "write manifest to PATH instead of stdout; with no PATH writes to $GORMES_HOME/slack-manifest.json")
	cmd.Flags().Lookup("write").NoOptDefVal = appslack.ManifestDefaultWrite
	cmd.Flags().StringVar(&opts.BotName, "name", "", `bot display name (default: "Gormes")`)
	cmd.Flags().StringVar(&opts.Description, "description", "", "bot description shown in Slack's app directory")
	cmd.Flags().BoolVar(&opts.SlashesOnly, "slashes-only", false, "emit only the features.slash_commands array")
	return cmd
}

func RunSlackManifestCommand(cmd *cobra.Command, opts SlackManifestOptions) error {
	return appslack.RunManifest(cmd.OutOrStdout(), cmd.ErrOrStderr(), appslack.ManifestOptions{
		BotName:      opts.BotName,
		Description:  opts.Description,
		SlashesOnly:  opts.SlashesOnly,
		WriteChanged: opts.WriteChanged,
		WriteTarget:  opts.WriteTarget,
	})
}

func SlackManifestPayload(botName, description string, slashesOnly bool) (any, error) {
	return appslack.ManifestPayload(botName, description, slashesOnly)
}

func SlackManifestSlashCommands(requestURL string) []map[string]any {
	return appslack.ManifestSlashCommands(requestURL)
}

func SanitizeSlackManifestName(name string) string { return appslack.SanitizeManifestName(name) }

func ClampSlackString(s string, max int) string { return appslack.ClampString(s, max) }

func NonEmptySlackValue(value, fallback string) string { return appslack.NonEmpty(value, fallback) }

func ExpandSlackUserPath(path string) string { return appslack.ExpandUserPath(path) }

func WriteSlackFileAtomic(path string, body []byte, perm fs.FileMode) error {
	return appslack.WriteFileAtomic(path, body, perm)
}
