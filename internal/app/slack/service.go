package slack

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const ManifestDefaultWrite = "__gormes_default_slack_manifest_path__"

type ManifestOptions struct {
	BotName      string
	Description  string
	SlashesOnly  bool
	WriteChanged bool
	WriteTarget  string
}

var manifestInvalidChars = regexp.MustCompile(`[^a-z0-9_-]`)

var manifestReservedCommands = map[string]struct{}{
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

func RunManifest(out, errOut io.Writer, opts ManifestOptions) error {
	payload, err := ManifestPayload(opts.BotName, opts.Description, opts.SlashesOnly)
	if err != nil {
		return err
	}
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("encode slack manifest: %w", err)
	}
	body = append(body, '\n')

	if !opts.WriteChanged {
		_, err = out.Write(body)
		return err
	}

	target := opts.WriteTarget
	if target == "" || target == ManifestDefaultWrite {
		target = filepath.Join(config.GormesHome(), "slack-manifest.json")
	}
	target = ExpandUserPath(target)
	if err := WriteFileAtomic(target, body, 0o600); err != nil {
		return fmt.Errorf("write slack manifest: %w", err)
	}
	fmt.Fprintf(errOut, "Slack manifest written to: %s\n\n", target)
	fmt.Fprintln(errOut, "Next steps:")
	fmt.Fprintln(errOut, "  1. Open https://api.slack.com/apps and pick your Gormes app.")
	fmt.Fprintf(errOut, "  2. Features -> App Manifest -> paste the contents of\n     %s\n", target)
	fmt.Fprintln(errOut, "  3. Save; Slack will prompt to reinstall the app if scopes or slash commands changed.")
	fmt.Fprintln(errOut, "  4. Make sure Socket Mode is enabled and bot/app tokens are configured with `gormes setup gateway`.")
	return nil
}

func ManifestPayload(botName, description string, slashesOnly bool) (any, error) {
	slashes := ManifestSlashCommands("https://gormes-agent.local/slack/commands")
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
			"name":             ClampString(name, 35),
			"description":      ClampString(desc, 140),
			"background_color": "#1a1a2e",
		},
		"features": map[string]any{
			"app_home": map[string]any{
				"home_tab_enabled":               false,
				"messages_tab_enabled":           true,
				"messages_tab_read_only_enabled": false,
			},
			"bot_user": map[string]any{
				"display_name":  ClampString(name, 80),
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

func ManifestSlashCommands(requestURL string) []map[string]any {
	const maxSlackCommands = 50

	out := make([]map[string]any, 0, maxSlackCommands)
	seen := map[string]struct{}{}
	add := func(name, description, usage string) {
		if len(out) >= maxSlackCommands {
			return
		}
		name = SanitizeManifestName(name)
		if name == "" {
			return
		}
		if _, reserved := manifestReservedCommands[name]; reserved {
			return
		}
		if _, exists := seen[name]; exists {
			return
		}
		seen[name] = struct{}{}
		entry := map[string]any{
			"command":       "/" + name,
			"description":   ClampString(NonEmpty(description, "Run /"+name), 140),
			"should_escape": false,
			"url":           requestURL,
		}
		if strings.TrimSpace(usage) != "" {
			entry["usage_hint"] = ClampString(usage, 100)
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

func SanitizeManifestName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = manifestInvalidChars.ReplaceAllString(name, "")
	name = strings.Trim(name, "-_")
	return ClampString(name, 32)
}

func ClampString(s string, max int) string {
	if max < 0 || len(s) <= max {
		return s
	}
	return s[:max]
}

func NonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func ExpandUserPath(path string) string {
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

func WriteFileAtomic(path string, body []byte, perm os.FileMode) error {
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
