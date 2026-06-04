package main

import (
	"os"

	"github.com/spf13/cobra"

	slackcmd "github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/slack"
	channelsmodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/channels"
)

func newSlackCommand() *cobra.Command {
	return slackcmd.NewCommand()
}

func runSlackManifestCommand(cmd *cobra.Command, opts channelsmodule.SlackManifestOptions) error {
	return slackcmd.RunManifestCommand(cmd, opts)
}

func slackManifestPayload(botName, description string, slashesOnly bool) (any, error) {
	return slackcmd.ManifestPayload(botName, description, slashesOnly)
}

func slackManifestSlashCommands(requestURL string) []map[string]any {
	return slackcmd.ManifestSlashCommands(requestURL)
}

func sanitizeSlackManifestName(name string) string {
	return slackcmd.SanitizeManifestName(name)
}

func clampString(s string, max int) string {
	return slackcmd.ClampString(s, max)
}

func nonEmpty(value, fallback string) string {
	return slackcmd.NonEmpty(value, fallback)
}

func expandUserPath(path string) string {
	return slackcmd.ExpandUserPath(path)
}

func writeFileAtomic(path string, body []byte, perm os.FileMode) error {
	return slackcmd.WriteFileAtomic(path, body, perm)
}
