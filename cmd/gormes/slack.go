package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	channelsmodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/channels"
)

func newSlackCommand() *cobra.Command {
	return channelsmodule.NewSlackCommandWithSeams(channelsmodule.SlackCommandSeams{
		Manifest: runSlackManifestCommand,
	})
}

func runSlackManifestCommand(cmd *cobra.Command, opts channelsmodule.SlackManifestOptions) error {
	payload, err := slackManifestPayload(opts.BotName, opts.Description, opts.SlashesOnly)
	if err != nil {
		return err
	}
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("encode slack manifest: %w", err)
	}
	body = append(body, '\n')

	if !opts.WriteChanged {
		_, err = cmd.OutOrStdout().Write(body)
		return err
	}

	target := opts.WriteTarget
	if target == "" || target == channelsmodule.SlackManifestDefaultWrite {
		target = filepath.Join(config.GormesHome(), "slack-manifest.json")
	}
	target = expandUserPath(target)
	if err := writeFileAtomic(target, body, 0o600); err != nil {
		return fmt.Errorf("write slack manifest: %w", err)
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "Slack manifest written to: %s\n\n", target)
	fmt.Fprintln(cmd.ErrOrStderr(), "Next steps:")
	fmt.Fprintln(cmd.ErrOrStderr(), "  1. Open https://api.slack.com/apps and pick your Gormes app.")
	fmt.Fprintf(cmd.ErrOrStderr(), "  2. Features -> App Manifest -> paste the contents of\n     %s\n", target)
	fmt.Fprintln(cmd.ErrOrStderr(), "  3. Save; Slack will prompt to reinstall the app if scopes or slash commands changed.")
	fmt.Fprintln(cmd.ErrOrStderr(), "  4. Make sure Socket Mode is enabled and bot/app tokens are configured with `gormes setup gateway`.")
	return nil
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
	return gormescli.ClampString(s, max)
}

func nonEmpty(value, fallback string) string {
	return gormescli.NonEmpty(value, fallback)
}

func expandUserPath(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}

func writeFileAtomic(path string, body []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
