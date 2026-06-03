package slack

import (
	"regexp"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

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
