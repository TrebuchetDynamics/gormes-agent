package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	channelsmodule "github.com/TrebuchetDynamics/gormes-agent/internal/app/gormescli/modules/channels"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

var slackManifestInvalidChars = regexp.MustCompile(`[^a-z0-9_-]`)

var slackManifestReservedCommands = map[string]struct{}{
	"away":      {},
	"collapse":  {},
	"dnd":       {},
	"expand":    {},
	"feed":      {},
	"join":      {},
	"leave":     {},
	"me":        {},
	"msg":       {},
	"mute":      {},
	"open":      {},
	"pro":       {},
	"remind":    {},
	"search":    {},
	"shortcuts": {},
	"shrug":     {},
	"status":    {},
	"topic":     {},
	"who":       {},
}

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
	slashes := slackManifestSlashCommands("https://gormes-agent.local/slack/commands")
	if slashesOnly {
		return slashes, nil
	}

	name := strings.TrimSpace(botName)
	if name == "" {
		name = "Gormes"
	}
	desc := strings.TrimSpace(description)
	if desc == "" {
		desc = "Your Gormes agent on Slack"
	}

	return map[string]any{
		"_metadata": map[string]any{
			"major_version": 1,
			"minor_version": 1,
		},
		"display_information": map[string]any{
			"name":             clampString(name, 35),
			"description":      clampString(desc, 140),
			"background_color": "#1a1a2e",
		},
		"features": map[string]any{
			"app_home": map[string]any{
				"home_tab_enabled":               false,
				"messages_tab_enabled":           true,
				"messages_tab_read_only_enabled": false,
			},
			"bot_user": map[string]any{
				"display_name":  clampString(name, 80),
				"always_online": true,
			},
			"slash_commands": slashes,
			"assistant_view": map[string]any{
				"assistant_description": "Chat with Gormes in threads and DMs.",
			},
		},
		"oauth_config": map[string]any{
			"scopes": map[string]any{
				"bot": []string{
					"app_mentions:read",
					"assistant:write",
					"channels:history",
					"channels:read",
					"chat:write",
					"commands",
					"files:read",
					"files:write",
					"groups:history",
					"groups:read",
					"im:history",
					"im:read",
					"im:write",
					"users:read",
				},
			},
		},
		"settings": map[string]any{
			"event_subscriptions": map[string]any{
				"bot_events": []string{
					"app_mention",
					"assistant_thread_context_changed",
					"assistant_thread_started",
					"message.channels",
					"message.groups",
					"message.im",
				},
			},
			"interactivity": map[string]any{
				"is_enabled": true,
			},
			"org_deploy_enabled":     false,
			"socket_mode_enabled":    true,
			"token_rotation_enabled": false,
		},
	}, nil
}

func slackManifestSlashCommands(requestURL string) []map[string]any {
	const maxSlackCommands = 50

	out := make([]map[string]any, 0, maxSlackCommands)
	seen := map[string]struct{}{}
	add := func(name, description, usage string) {
		if len(out) >= maxSlackCommands {
			return
		}
		name = sanitizeSlackManifestName(name)
		if name == "" {
			return
		}
		if _, reserved := slackManifestReservedCommands[name]; reserved {
			return
		}
		if _, exists := seen[name]; exists {
			return
		}
		seen[name] = struct{}{}
		entry := map[string]any{
			"command":       "/" + name,
			"description":   clampString(nonEmpty(description, "Run /"+name), 140),
			"should_escape": false,
			"url":           requestURL,
		}
		if strings.TrimSpace(usage) != "" {
			entry["usage_hint"] = clampString(usage, 100)
		}
		out = append(out, entry)
	}

	add("hermes", "Talk to Gormes or run a subcommand", "[subcommand] [args]")
	for _, cmd := range gateway.CommandRegistry {
		add(cmd.Name, cmd.Description, "")
	}
	for _, cmd := range gateway.CommandRegistry {
		for _, alias := range cmd.Aliases {
			add(alias, "Alias for /"+cmd.Name+" - "+cmd.Description, "")
		}
	}
	return out
}

func sanitizeSlackManifestName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = slackManifestInvalidChars.ReplaceAllString(name, "")
	name = strings.Trim(name, "-_")
	return clampString(name, 32)
}

func clampString(s string, max int) string {
	if max < 0 || len(s) <= max {
		return s
	}
	return s[:max]
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
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
