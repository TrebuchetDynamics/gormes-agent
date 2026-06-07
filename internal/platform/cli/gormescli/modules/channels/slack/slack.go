package slack

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/channels/internal/seam"
)

const SlackManifestDefaultWrite = "__gormes_default_slack_manifest_path__"

type SlackManifestOptions struct {
	BotName      string
	Description  string
	SlashesOnly  bool
	WriteTarget  string
	WriteChanged bool
}

type SlackCommandSeams struct {
	Manifest func(*cobra.Command, SlackManifestOptions) error
}

func NewSlackCommandWithSeams(seams SlackCommandSeams) *cobra.Command {
	seams = seams.withDefaults()
	cmd := &cobra.Command{
		Use:   "slack",
		Short: "Slack integration helpers",
	}
	cmd.AddCommand(newSlackManifestCommand(seams))
	return cmd
}

func (s SlackCommandSeams) withDefaults() SlackCommandSeams {
	if s.Manifest == nil {
		s.Manifest = func(*cobra.Command, SlackManifestOptions) error {
			return seam.Missing("slack manifest")
		}
	}
	return s
}

func newSlackManifestCommand(seams SlackCommandSeams) *cobra.Command {
	opts := SlackManifestOptions{}
	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "Print or write a Slack app manifest",
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts.WriteChanged = cmd.Flags().Changed("write")
			return seams.Manifest(cmd, opts)
		},
	}
	cmd.Flags().StringVar(&opts.WriteTarget, "write", "", "write manifest to PATH instead of stdout; with no PATH writes to $GORMES_HOME/slack-manifest.json")
	cmd.Flags().Lookup("write").NoOptDefVal = SlackManifestDefaultWrite
	cmd.Flags().StringVar(&opts.BotName, "name", "", `bot display name (default: "Gormes")`)
	cmd.Flags().StringVar(&opts.Description, "description", "", "bot description shown in Slack's app directory")
	cmd.Flags().BoolVar(&opts.SlashesOnly, "slashes-only", false, "emit only the features.slash_commands array")
	return cmd
}
