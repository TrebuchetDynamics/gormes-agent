package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

func newSlackCommand() *cobra.Command {
	return gormescli.NewSlackCommand()
}

func runSlackManifestCommand(cmd *cobra.Command, opts gormescli.SlackManifestOptions) error {
	return gormescli.RunSlackManifestCommand(cmd, opts)
}

func slackManifestPayload(botName, description string, slashesOnly bool) (any, error) {
	return gormescli.SlackManifestPayload(botName, description, slashesOnly)
}

func slackManifestSlashCommands(requestURL string) []map[string]any {
	return gormescli.SlackManifestSlashCommands(requestURL)
}

func sanitizeSlackManifestName(name string) string {
	return gormescli.SanitizeSlackManifestName(name)
}

func clampString(s string, max int) string {
	return gormescli.ClampSlackString(s, max)
}

func nonEmpty(value, fallback string) string {
	return gormescli.NonEmptySlackValue(value, fallback)
}

func expandUserPath(path string) string {
	return gormescli.ExpandSlackUserPath(path)
}

func writeFileAtomic(path string, body []byte, perm os.FileMode) error {
	return gormescli.WriteSlackFileAtomic(path, body, perm)
}
