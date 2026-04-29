package main

import (
	"slices"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
)

type hermesCLIParityStatus string

const (
	hermesCLIImplemented hermesCLIParityStatus = "implemented"
	hermesCLIRowBacked   hermesCLIParityStatus = "row_backed"
	hermesCLIOwned       hermesCLIParityStatus = "owned"
	hermesCLIExcluded    hermesCLIParityStatus = "excluded"
)

type hermesCLIParityKind string

const (
	hermesCLICommand        hermesCLIParityKind = "command"
	hermesCLICommandSet     hermesCLIParityKind = "command_set"
	hermesCLIGlobalFlag     hermesCLIParityKind = "global_flag"
	hermesCLISlashCommand   hermesCLIParityKind = "slash_command"
	hermesCLIAlias          hermesCLIParityKind = "alias"
	hermesCLIGatewayHandler hermesCLIParityKind = "gateway_handler"
	hermesCLIPluginCommand  hermesCLIParityKind = "plugin_command"
)

type hermesCLIParityEntry struct {
	Path           []string              `json:"path"`
	Kind           hermesCLIParityKind   `json:"kind"`
	Status         hermesCLIParityStatus `json:"status"`
	SourceRef      string                `json:"source_ref"`
	Target         string                `json:"target,omitempty"`
	Residual       string                `json:"residual,omitempty"`
	Row            string                `json:"row,omitempty"`
	AliasFor       []string              `json:"alias_for,omitempty"`
	Dynamic        bool                  `json:"dynamic,omitempty"`
	Destructive    bool                  `json:"destructive,omitempty"`
	DryRun         bool                  `json:"dry_run,omitempty"`
	RedactsSecrets bool                  `json:"redacts_secrets,omitempty"`
}

func hermesCLIParityManifest() []hermesCLIParityEntry {
	entries := []hermesCLIParityEntry{
		hermesImplementedCommand("chat", "hermes_cli/main.py:chat", "cmd/gormes root TUI/oneshot"),
		hermesRowCommand("model", "hermes_cli/main.py:model_command", "Gormes model interactive provider/model picker", "interactive model picker and account-aware provider switching remain row-backed"),
		hermesCommandSet("fallback", "hermes_cli/main.py:fallback", "fallback provider chain handlers remain row-backed", "Hermes fallback provider chain CLI commands"),
		hermesCommandSet("gateway", "hermes_cli/main.py:gateway", "gateway lifecycle subcommands are partly implemented; missing mutating/service commands remain row-backed", "Gateway, platform, webhook, and cron management CLI"),
		hermesRowCommand("setup", "hermes_cli/main.py:setup", "Gormes config command surface", "interactive setup wizard remains row-backed; current config is TOML/env loaded non-interactively"),
		hermesRowCommand("whatsapp", "hermes_cli/main.py:whatsapp", "Gateway, platform, webhook, and cron management CLI", "WhatsApp platform management remains row-backed"),
		hermesRowCommand("slack", "hermes_cli/main.py:slack", "Gateway, platform, webhook, and cron management CLI", "Slack platform management remains row-backed"),
		hermesExcludedCommand("login", "hermes_cli/auth.py:login_command", "Hermes top-level login is removed; use `gormes auth add <provider> --type oauth`, `gormes model`, or `gormes setup` parity rows"),
		hermesProviderLogoutCommand(),
		hermesCommandSet("auth", "hermes_cli/auth_commands.py:auth_command", "provider auth subcommands remain row-backed", "Hermes auth credential-pool command surface"),
		hermesImplementedCommand("status", "hermes_cli/main.py:status", "cmd/gormes gateway status"),
		hermesCommandSet("cron", "hermes_cli/main.py:cron", "cron command handlers remain row-backed over native cron store", "Gateway, platform, webhook, and cron management CLI"),
		hermesCommandSet("webhook", "hermes_cli/main.py:webhook", "webhook management remains row-backed", "Gateway, platform, webhook, and cron management CLI"),
		hermesCommandSet("hooks", "hermes_cli/main.py:hooks", "local hook list/run surfaces remain row-backed", "Gateway, platform, webhook, and cron management CLI"),
		hermesImplementedCommand("doctor", "hermes_cli/main.py:doctor", "cmd/gormes doctor"),
		hermesRowCommand("dump", "hermes_cli/main.py:dump", "Diagnostics, backup, logs, and status CLI", "debug dump output remains row-backed"),
		hermesCommandSet("debug", "hermes_cli/main.py:debug", "debug share/doctor helpers remain row-backed", "Diagnostics, backup, logs, and status CLI"),
		hermesRowCommand("backup", "hermes_cli/main.py:backup", "Backup/update opt-in and exclusion policy", "backup archive creation and exclusions remain row-backed"),
		hermesRowCommand("import", "hermes_cli/main.py:import", "Hermes config migration dry-run manifest", "Hermes import/migration command remains row-backed"),
		hermesCommandSet("config", "hermes_cli/config.py", "config show/set/check/edit/migrate handlers remain row-backed", "Gormes config command surface"),
		hermesCommandSet("pairing", "hermes_cli/main.py:pairing", "pairing CLI management remains row-backed", "Gateway, platform, webhook, and cron management CLI"),
		hermesImplementedCommand("skills", "hermes_cli/main.py:skills", "cmd/gormes skills-compatible registry rows"),
		hermesCommandSet("plugins", "hermes_cli/plugins_cmd.py", "plugin manager command handlers are manifest-only until plugin runtime rows land", "Plugin SDK"),
		hermesImplementedCommand("memory", "plugins/memory/__init__.py:discover_plugin_cli_commands", "cmd/gormes memory"),
		hermesRowCommand("tools", "hermes_cli/main.py:tools", "Tool/runtime/security rows", "tool inventory and doctor commands remain row-backed"),
		hermesRowCommand("mcp", "hermes_cli/main.py:mcp", "ACP server side", "MCP management commands remain row-backed"),
		hermesImplementedCommand("sessions", "hermes_cli/main.py:sessions", "cmd/gormes session"),
		hermesRowCommand("insights", "hermes_cli/main.py:insights", "Self-monitoring telemetry", "insights rollup command remains row-backed"),
		hermesCommandSet("claw", "hermes_cli/claw.py", "OpenClaw migration commands remain row-backed", "OpenClaw migration dry-run manifest"),
		hermesImplementedCommand("version", "hermes_cli/main.py:version", "cmd/gormes version"),
		hermesRowCommand("update", "gateway/run.py:_handle_update_command", "Backup/update opt-in and exclusion policy", "self-update command remains row-backed"),
		hermesRowCommand("uninstall", "hermes_cli/main.py:uninstall", "Backup/update opt-in and exclusion policy", "uninstaller remains row-backed and destructive"),
		hermesRowCommand("acp", "hermes_cli/main.py:acp", "ACP server side", "ACP server/client command remains row-backed"),
		hermesRowCommand("profile", "hermes_cli/main.py:profile", "Gormes config command surface", "profile command remains row-backed"),
		hermesRowCommand("completion", "hermes_cli/main.py:completion", "Hermes CLI command-tree parity manifest", "shell completion command remains manifest-classified only"),
		hermesRowCommand("dashboard", "hermes_cli/main.py:dashboard", "Dashboard theme/plugin extension status contract", "dashboard launch/status command remains row-backed"),
		hermesRowCommand("logs", "hermes_cli/main.py:logs", "Diagnostics, backup, logs, and status CLI", "log snapshot command remains row-backed"),
		hermesOwnedCommand("goncho", "cmd/gormes/goncho.go", "Gormes-owned Honcho-compatible local memory namespace"),
		hermesGlobalFlag("-z", "hermes_cli/main.py:--oneshot", hermesCLIImplemented, "cmd/gormes --oneshot", "Hermes oneshot short flag parity"),
		hermesGlobalFlag("--oneshot", "hermes_cli/main.py:--oneshot", hermesCLIImplemented, "cmd/gormes --oneshot", "Hermes oneshot flag parity"),
		hermesGlobalFlag("--model", "hermes_cli/main.py:--model", hermesCLIImplemented, "cmd/gormes --model", "model override is implemented for startup resolution"),
		hermesGlobalFlag("--provider", "hermes_cli/main.py:--provider", hermesCLIImplemented, "cmd/gormes --provider", "provider override is implemented for startup resolution"),
		hermesGlobalFlag("--offline", "cmd/gormes/main.go:--offline", hermesCLIOwned, "cmd/gormes --offline", "Gormes-owned local smoke-test mode"),
		hermesGlobalFlag("--remote", "cmd/gormes/main.go:--remote", hermesCLIOwned, "cmd/gormes --remote", "Gormes-owned remote TUI connection mode"),
		hermesRowPath([]string{"migrate", "ooenclaw"}, hermesCLICommand, "operator request compatibility", "OpenClaw migration dry-run manifest", "must suggest `gormes migrate openclaw`; must not become a silent import alias"),
	}

	entries = append(entries, hermesNestedCommands("gateway", "gateway/run.py", "Gateway, platform, webhook, and cron management CLI", []string{"status", "restart", "reset", "help", "model", "profile", "update", "approve", "deny", "voice", "usage"})...)
	entries = append(entries, hermesFallbackCommands()...)
	entries = append(entries, hermesProviderAuthCommands()...)
	entries = append(entries, hermesNestedCommands("cron", "hermes_cli/main.py:cron", "Gateway, platform, webhook, and cron management CLI", []string{"list", "add", "remove", "run", "enable", "disable"})...)
	entries = append(entries, hermesNestedCommands("webhook", "hermes_cli/main.py:webhook", "Gateway, platform, webhook, and cron management CLI", []string{"serve", "test", "list", "add", "remove"})...)
	entries = append(entries, hermesNestedCommands("hooks", "hermes_cli/main.py:hooks", "Gateway, platform, webhook, and cron management CLI", []string{"list", "run"})...)
	entries = append(entries, hermesNestedCommands("debug", "hermes_cli/main.py:debug", "Diagnostics, backup, logs, and status CLI", []string{"doctor", "share", "paste", "sweep"})...)
	entries = append(entries, hermesNestedCommands("config", "hermes_cli/config.py", "Gormes config command surface", []string{"show", "set", "check", "edit", "migrate", "path"})...)
	entries = append(entries, hermesNestedCommands("pairing", "hermes_cli/main.py:pairing", "Gateway, platform, webhook, and cron management CLI", []string{"approve", "deny", "list", "reset"})...)
	entries = append(entries, hermesNestedCommands("skills", "hermes_cli/main.py:skills", "Skills hub direct URL install name/category guard", []string{"list", "search", "install", "remove", "tap", "snapshot", "check"})...)
	entries = append(entries, hermesNestedCommands("plugins", "hermes_cli/plugins_cmd.py", "Plugin SDK", []string{"list", "enable", "disable", "install", "remove", "doctor"})...)
	entries = append(entries, hermesNestedCommands("memory", "plugins/memory/__init__.py", "Goncho memory integration into normal agent turn", []string{"search", "add", "status", "delete", "export"})...)
	entries = append(entries, hermesNestedCommands("tools", "hermes_cli/main.py:tools", "Tool/runtime/security rows", []string{"list", "doctor", "enable", "disable"})...)
	entries = append(entries, hermesNestedCommands("mcp", "hermes_cli/main.py:mcp", "ACP server side", []string{"list", "call", "add", "remove", "auth"})...)
	entries = append(entries, hermesNestedCommands("sessions", "hermes_cli/main.py:sessions", "Session shutdown memory transcript handoff", []string{"list", "resume", "export", "delete", "rename"})...)
	entries = append(entries, hermesNestedCommands("claw", "hermes_cli/claw.py", "OpenClaw migration dry-run manifest", []string{"migrate", "cleanup"})...)
	entries = append(entries, hermesNestedCommands("profile", "hermes_cli/main.py:profile", "Gormes config command surface", []string{"show", "set", "list"})...)

	entries = append(entries, hermesGatewayHandlers()...)
	entries = append(entries, hermesSlashRegistryEntries()...)
	entries = append(entries,
		hermesDynamicPluginCommand("memory", "plugins/memory/__init__.py:discover_plugin_cli_commands", hermesCLIImplemented, "cmd/gormes memory", "memory plugin CLI is implemented as a first-party Gormes memory command surface"),
		hermesDynamicPluginCommand("disk-cleanup", "plugins/disk-cleanup/__init__.py:PluginContext.register_command", hermesCLIRowBacked, "", "disk cleanup dynamic plugin command remains row-backed under Plugin SDK/tool security rows"),
	)
	return cloneHermesCLIParityEntries(entries)
}

func hermesImplementedCommand(name, sourceRef, target string) hermesCLIParityEntry {
	return hermesCLIParityEntry{Path: []string{name}, Kind: hermesCLICommand, Status: hermesCLIImplemented, SourceRef: sourceRef, Target: target, Residual: "implemented or classified by Gormes command surface tests"}
}

func hermesRowCommand(name, sourceRef, row, residual string) hermesCLIParityEntry {
	return hermesCLIParityEntry{Path: []string{name}, Kind: hermesCLICommand, Status: hermesCLIRowBacked, SourceRef: sourceRef, Row: row, Residual: residual}
}

func hermesCommandSet(name, sourceRef, residual, row string) hermesCLIParityEntry {
	return hermesCLIParityEntry{Path: []string{name}, Kind: hermesCLICommandSet, Status: hermesCLIRowBacked, SourceRef: sourceRef, Row: row, Residual: residual}
}

func hermesOwnedCommand(name, sourceRef, residual string) hermesCLIParityEntry {
	return hermesCLIParityEntry{Path: []string{name}, Kind: hermesCLICommand, Status: hermesCLIOwned, SourceRef: sourceRef, Target: "cmd/gormes " + name, Residual: residual}
}

func hermesExcludedCommand(name, sourceRef, residual string) hermesCLIParityEntry {
	return hermesCLIParityEntry{Path: []string{name}, Kind: hermesCLICommand, Status: hermesCLIExcluded, SourceRef: sourceRef, Residual: residual}
}

func hermesProviderLogoutCommand() hermesCLIParityEntry {
	entry := hermesRowCommand("logout", "hermes_cli/auth.py:logout_command", "Hermes provider auth CLI commands", "top-level provider logout remains row-backed; clears auth state and resets provider config")
	entry.RedactsSecrets = true
	entry.Destructive = true
	return entry
}

func hermesGlobalFlag(flag, sourceRef string, status hermesCLIParityStatus, target, residual string) hermesCLIParityEntry {
	return hermesCLIParityEntry{Path: []string{flag}, Kind: hermesCLIGlobalFlag, Status: status, SourceRef: sourceRef, Target: target, Residual: residual}
}

func hermesRowPath(path []string, kind hermesCLIParityKind, sourceRef, row, residual string) hermesCLIParityEntry {
	return hermesCLIParityEntry{Path: slices.Clone(path), Kind: kind, Status: hermesCLIRowBacked, SourceRef: sourceRef, Row: row, Residual: residual}
}

func hermesNestedCommands(group, sourceRef, row string, commands []string) []hermesCLIParityEntry {
	out := make([]hermesCLIParityEntry, 0, len(commands))
	for _, command := range commands {
		entry := hermesRowPath([]string{group, command}, hermesCLICommand, sourceRef+":"+command, row, group+" "+command+" handler remains classified by this manifest; implementation lands in its dedicated row")
		key := strings.Join(entry.Path, " ")
		switch key {
		case "backup create", "sessions delete", "claw cleanup", "config set", "plugins remove", "skills remove", "pairing reset":
			entry.Destructive = true
		}
		switch key {
		case "config set", "auth login", "login":
			entry.RedactsSecrets = true
		}
		switch key {
		case "claw migrate", "config migrate":
			entry.DryRun = true
		}
		out = append(out, entry)
	}
	return out
}

func hermesProviderAuthCommands() []hermesCLIParityEntry {
	commands := []struct {
		name        string
		sourceRef   string
		residual    string
		status      hermesCLIParityStatus
		target      string
		destructive bool
		redacts     bool
	}{
		{
			name:      "add",
			sourceRef: "hermes_cli/auth_commands.py:auth_add_command",
			residual:  "non-deprecated provider login/add flow is partially implemented: API keys and openai-codex device-code OAuth are native; anthropic, nous, qwen-oauth, and google-gemini-cli OAuth remain row-backed",
			redacts:   true,
		},
		{
			name:      "list",
			sourceRef: "hermes_cli/auth_commands.py:auth_list_command",
			residual:  "redacted credential-pool listing is implemented over the native auth.json pool",
			status:    hermesCLIImplemented,
			target:    "cmd/gormes auth list",
		},
		{
			name:        "remove",
			sourceRef:   "hermes_cli/auth_commands.py:auth_remove_command",
			residual:    "credential removal by index, id, or label is implemented; source suppression is reported as not_applicable for manual credentials",
			status:      hermesCLIImplemented,
			target:      "cmd/gormes auth remove",
			destructive: true,
			redacts:     true,
		},
		{
			name:      "reset",
			sourceRef: "hermes_cli/auth_commands.py:auth_reset_command",
			residual:  "provider credential exhaustion/cooldown reset is implemented over the native auth.json pool",
			status:    hermesCLIImplemented,
			target:    "cmd/gormes auth reset",
		},
		{
			name:      "status",
			sourceRef: "hermes_cli/auth_commands.py:auth_status_command",
			residual:  "provider auth status read model is implemented over the native auth.json pool; provider-specific OAuth status expansion remains in adapter rows",
			status:    hermesCLIImplemented,
			target:    "cmd/gormes auth status",
		},
		{
			name:        "logout",
			sourceRef:   "hermes_cli/auth_commands.py:auth_logout_command",
			residual:    "provider logout clears native credential-pool entries; top-level logout shortcut remains row-backed",
			status:      hermesCLIImplemented,
			target:      "cmd/gormes auth logout",
			destructive: true,
			redacts:     true,
		},
		{
			name:      "spotify",
			sourceRef: "hermes_cli/auth_commands.py:auth_spotify_command",
			residual:  "Spotify PKCE auth actions login|status|logout remain row-backed under provider auth CLI parity",
			redacts:   true,
		},
	}
	out := make([]hermesCLIParityEntry, 0, len(commands))
	for _, command := range commands {
		entry := hermesRowPath([]string{"auth", command.name}, hermesCLICommand, command.sourceRef, "Hermes provider auth CLI commands", command.residual)
		if command.status != "" {
			entry.Status = command.status
			entry.Target = command.target
		}
		entry.Destructive = command.destructive
		entry.RedactsSecrets = command.redacts
		out = append(out, entry)
	}
	return out
}

func hermesFallbackCommands() []hermesCLIParityEntry {
	commands := []struct {
		path      []string
		sourceRef string
		residual  string
		aliasFor  []string
	}{
		{path: []string{"fallback", "list"}, sourceRef: "hermes_cli/fallback_cmd.py:cmd_fallback_list", residual: "fallback chain list renderer remains row-backed"},
		{path: []string{"fallback", "ls"}, sourceRef: "hermes_cli/main.py:fallback aliases", residual: "fallback ls alias resolves to fallback list", aliasFor: []string{"fallback", "list"}},
		{path: []string{"fallback", "add"}, sourceRef: "hermes_cli/fallback_cmd.py:cmd_fallback_add", residual: "fallback add uses the Hermes model picker then appends provider/model to fallback_providers"},
		{path: []string{"fallback", "remove"}, sourceRef: "hermes_cli/fallback_cmd.py:cmd_fallback_remove", residual: "fallback remove deletes a selected provider/model fallback entry", aliasFor: nil},
		{path: []string{"fallback", "rm"}, sourceRef: "hermes_cli/main.py:fallback aliases", residual: "fallback rm alias resolves to fallback remove", aliasFor: []string{"fallback", "remove"}},
		{path: []string{"fallback", "clear"}, sourceRef: "hermes_cli/fallback_cmd.py:cmd_fallback_clear", residual: "fallback clear removes all fallback provider entries after confirmation"},
	}
	out := make([]hermesCLIParityEntry, 0, len(commands))
	for _, command := range commands {
		kind := hermesCLICommand
		if len(command.aliasFor) > 0 {
			kind = hermesCLIAlias
		}
		entry := hermesRowPath(command.path, kind, command.sourceRef, "Hermes fallback provider chain CLI commands", command.residual)
		entry.AliasFor = slices.Clone(command.aliasFor)
		if command.path[1] == "remove" || command.path[1] == "rm" || command.path[1] == "clear" {
			entry.Destructive = true
		}
		out = append(out, entry)
	}
	return out
}

func hermesGatewayHandlers() []hermesCLIParityEntry {
	handlers := []string{"status", "restart", "reset", "help", "model", "profile", "update", "approve", "deny", "voice", "usage"}
	out := make([]hermesCLIParityEntry, 0, len(handlers))
	for _, handler := range handlers {
		status := hermesCLIRowBacked
		target := ""
		row := "Gateway, platform, webhook, and cron management CLI"
		residual := "gateway handler remains row-backed or command-specific unless target is set"
		switch handler {
		case "status":
			status, target, residual = hermesCLIImplemented, "cmd/gormes gateway status", "read-only gateway status command is implemented"
		case "usage":
			status, target, residual = hermesCLIImplemented, "cmd/gormes usage and /usage", "provider account usage binding is implemented"
		case "help", "restart", "reset":
			status, target = hermesCLIImplemented, "internal/gateway command dispatcher"
		}
		out = append(out, hermesCLIParityEntry{Path: []string{"gateway-handler", handler}, Kind: hermesCLIGatewayHandler, Status: status, SourceRef: "gateway/run.py:_handle_" + strings.ReplaceAll(handler, "-", "_") + "_command", Target: target, Row: row, Residual: residual})
	}
	return out
}

func hermesSlashRegistryEntries() []hermesCLIParityEntry {
	var out []hermesCLIParityEntry
	for _, policy := range cli.CommandRegistry {
		status := hermesCLIRowBacked
		target := ""
		row := "slash command handler rows"
		if policy.Ported {
			status = hermesCLIImplemented
			target = "internal/cli.CommandRegistry /" + policy.Name
		}
		out = append(out, hermesCLIParityEntry{Path: []string{"/" + policy.Name}, Kind: hermesCLISlashCommand, Status: status, SourceRef: "hermes_cli/commands.py:COMMAND_REGISTRY", Target: target, Row: row, Residual: "slash command is cross-linked to internal/cli.CommandRegistry active-turn policy"})
		for _, alias := range policy.Aliases {
			out = append(out, hermesCLIParityEntry{Path: []string{"/" + alias}, Kind: hermesCLIAlias, Status: status, SourceRef: "hermes_cli/commands.py:COMMAND_REGISTRY aliases", Target: target, Row: row, AliasFor: []string{"/" + policy.Name}, Residual: "slash alias resolves to /" + policy.Name + " through the active-turn registry"})
		}
	}
	return out
}

func hermesDynamicPluginCommand(name, sourceRef string, status hermesCLIParityStatus, target, residual string) hermesCLIParityEntry {
	return hermesCLIParityEntry{Path: []string{"plugins", "dynamic", name}, Kind: hermesCLIPluginCommand, Status: status, SourceRef: sourceRef, Target: target, Row: "Plugin SDK", Residual: residual, Dynamic: true}
}

func cloneHermesCLIParityEntries(in []hermesCLIParityEntry) []hermesCLIParityEntry {
	out := make([]hermesCLIParityEntry, len(in))
	for i, entry := range in {
		out[i] = entry
		out[i].Path = slices.Clone(entry.Path)
		out[i].AliasFor = slices.Clone(entry.AliasFor)
	}
	return out
}
