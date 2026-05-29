package main

import (
	"slices"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
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
		hermesImplementedCommand("chat", "hermes_cli/main.py:chat", "cmd/gormes chat/root TUI"),
		hermesImplementedCommand("model", "hermes_cli/main.py:model_command", "internal/gateway model handler"),
		hermesImplementedCommand("fallback", "hermes_cli/main.py:fallback", "cmd/gormes fallback"),
		hermesCommandSet("gateway", "hermes_cli/main.py:gateway", "gateway lifecycle subcommands are partly implemented; missing mutating/service commands remain row-backed", "Gateway, platform, webhook, and cron management CLI"),
		hermesRowCommand("setup", "hermes_cli/main.py:setup", "Gormes config command surface", "interactive setup wizard remains row-backed; current config is TOML/env loaded non-interactively"),
		{Path: []string{"whatsapp"}, Kind: hermesCLICommand, Status: hermesCLIImplemented, SourceRef: "hermes_cli/main.py:whatsapp", Target: "cmd/gormes whatsapp", Row: "WhatsApp top-level pairing wizard shell", Residual: "top-level WhatsApp wizard shell and plan output are implemented; bundling/running the live Baileys QR bridge remains row-backed"},
		hermesRowCommand("slack", "hermes_cli/main.py:slack", "Gateway, platform, webhook, and cron management CLI", "Slack platform management remains row-backed"),
		hermesExcludedCommand("login", "hermes_cli/main.py:login_parser + hermes_cli/auth.py:_login_openai_codex", "removed top-level command; use `gormes auth add <provider> --type oauth`"),
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
		hermesCommandSet("kanban", "hermes_cli/kanban.py:build_parser", "durable board core is implemented in cmd/gormes; multi-board, dispatcher, worker-tool, notification, slash/gateway, and dashboard surfaces remain row-backed", "Hermes Kanban durable board core"),
		hermesImplementedCommand("send", "hermes_cli/send_cmd.py:register_send_subparser", "cmd/gormes send"),
		hermesCommandSet("claw", "hermes_cli/claw.py", "`claw migrate` and `claw cleanup` compatibility spellings are implemented over the Gormes-native OpenClaw migration engine", "OpenClaw migration dry-run manifest"),
		hermesImplementedCommand("version", "hermes_cli/main.py:version", "cmd/gormes version"),
		hermesImplementedCommand("curator", "hermes_cli/main.py:curator", "cmd/gormes curator"),
		hermesImplementedCommand("retry", "gateway/run.py:_handle_retry_command", "internal/gateway retry"),
		hermesImplementedCommand("platforms", "gateway/run.py:_handle_status_command", "internal/gateway platforms alias"),
		hermesRowCommand("update", "gateway/run.py:_handle_update_command", "Backup/update opt-in and exclusion policy", "self-update command remains row-backed"),
		hermesRowCommand("uninstall", "hermes_cli/main.py:uninstall", "Backup/update opt-in and exclusion policy", "uninstaller remains row-backed and destructive"),
		hermesRowCommand("acp", "hermes_cli/main.py:acp", "ACP server side", "ACP server/client command remains row-backed"),
		hermesImplementedCommand("profile", "hermes_cli/main.py:profile", "internal/gateway profile command handler"),
		hermesImplementedCommand("completion", "hermes_cli/main.py:completion", "cmd/gormes completion"),
		hermesRowCommand("dashboard", "hermes_cli/main.py:dashboard", "Dashboard theme/plugin extension status contract", "dashboard launch/status command remains row-backed"),
		hermesRowCommand("logs", "hermes_cli/main.py:logs", "Diagnostics, backup, logs, and status CLI", "log snapshot command remains row-backed"),
		hermesOwnedCommand("goncho", "cmd/gormes/goncho.go", "Gormes-owned Honcho-compatible local memory namespace"),
		hermesOwnedCommand("agent", "cmd/gormes/agent.go", "Gormes-owned agent context template reset command"),
		hermesGlobalFlag("-z", "hermes_cli/main.py:--oneshot", hermesCLIExcluded, "", "removed root flag; use `gormes chat -q`"),
		hermesGlobalFlag("--oneshot", "hermes_cli/main.py:--oneshot", hermesCLIExcluded, "", "removed root flag; use `gormes chat -q`"),
		hermesGlobalFlag("--model", "hermes_cli/main.py:--model", hermesCLIImplemented, "cmd/gormes --model", "model override is implemented for startup resolution"),
		hermesGlobalFlag("--provider", "hermes_cli/main.py:--provider", hermesCLIImplemented, "cmd/gormes --provider", "provider override is implemented for startup resolution"),
		hermesGlobalFlag("--offline", "cmd/gormes/main.go:--offline", hermesCLIOwned, "cmd/gormes --offline", "Gormes-owned local smoke-test mode"),
		hermesGlobalFlag("--remote", "cmd/gormes/main.go:--remote", hermesCLIOwned, "cmd/gormes --remote", "Gormes-owned remote TUI connection mode"),
		hermesRowPath([]string{"migrate", "ooenclaw"}, hermesCLICommand, "operator request compatibility", "OpenClaw migration dry-run manifest", "must suggest `gormes migrate openclaw`; must not become a silent import alias"),
	}

	entries = append(entries, hermesGatewayNestedCommands()...)
	entries = append(entries, hermesNestedCommands("slack", "hermes_cli/main.py:slack_sub", "Gateway, platform, webhook, and cron management CLI", []string{"manifest"})...)
	entries = append(entries, hermesFallbackCommands()...)
	entries = append(entries, hermesProviderAuthCommands()...)
	entries = append(entries, hermesNestedCommands("cron", "hermes_cli/main.py:cron_subparsers", "Gateway, platform, webhook, and cron management CLI", []string{"list", "create", "edit", "pause", "resume", "run", "remove", "status", "tick"})...)
	entries = append(entries,
		hermesNestedAlias("cron", "add", "create", "hermes_cli/main.py:cron_subparsers:create aliases", "Gateway, platform, webhook, and cron management CLI"),
		hermesNestedAlias("cron", "rm", "remove", "hermes_cli/main.py:cron_subparsers:remove aliases", "Gateway, platform, webhook, and cron management CLI"),
		hermesNestedAlias("cron", "delete", "remove", "hermes_cli/main.py:cron_subparsers:remove aliases", "Gateway, platform, webhook, and cron management CLI"),
	)
	entries = append(entries, hermesNestedCommands("webhook", "hermes_cli/main.py:webhook_subparsers", "Gateway, platform, webhook, and cron management CLI", []string{"subscribe", "list", "remove", "test"})...)
	entries = append(entries,
		hermesNestedAlias("webhook", "add", "subscribe", "hermes_cli/main.py:webhook_subparsers:subscribe aliases", "Gateway, platform, webhook, and cron management CLI"),
		hermesNestedAlias("webhook", "ls", "list", "hermes_cli/main.py:webhook_subparsers:list aliases", "Gateway, platform, webhook, and cron management CLI"),
		hermesNestedAlias("webhook", "rm", "remove", "hermes_cli/main.py:webhook_subparsers:remove aliases", "Gateway, platform, webhook, and cron management CLI"),
	)
	entries = append(entries, hermesNestedCommands("hooks", "hermes_cli/main.py:hooks_subparsers", "Gateway, platform, webhook, and cron management CLI", []string{"list", "test", "revoke", "doctor"})...)
	entries = append(entries,
		hermesNestedAlias("hooks", "ls", "list", "hermes_cli/main.py:hooks_subparsers:list aliases", "Gateway, platform, webhook, and cron management CLI"),
		hermesNestedAlias("hooks", "remove", "revoke", "hermes_cli/main.py:hooks_subparsers:revoke aliases", "Gateway, platform, webhook, and cron management CLI"),
		hermesNestedAlias("hooks", "rm", "revoke", "hermes_cli/main.py:hooks_subparsers:revoke aliases", "Gateway, platform, webhook, and cron management CLI"),
	)
	entries = append(entries, hermesNestedCommands("debug", "hermes_cli/main.py:debug_sub", "Diagnostics, backup, logs, and status CLI", []string{"share", "delete"})...)
	entries = append(entries, hermesNestedCommands("config", "hermes_cli/main.py:config_subparsers", "Gormes config command surface", []string{"show", "edit", "set", "path", "env-path", "check", "migrate"})...)
	entries = append(entries, hermesNestedCommands("pairing", "hermes_cli/main.py:pairing_sub", "Gateway, platform, webhook, and cron management CLI", []string{"list", "approve", "revoke", "clear-pending"})...)
	entries = append(entries, hermesNestedCommands("skills", "hermes_cli/main.py:skills_subparsers", "Skills hub direct URL install name/category guard", []string{"browse", "search", "install", "inspect", "list", "check", "update", "audit", "uninstall", "reset", "publish", "snapshot", "tap", "config"})...)
	entries = append(entries,
		hermesNestedSubcommand("skills", "snapshot", "export", "hermes_cli/main.py:snapshot_subparsers:export", "Skills hub direct URL install name/category guard"),
		hermesNestedSubcommand("skills", "snapshot", "import", "hermes_cli/main.py:snapshot_subparsers:import", "Skills hub direct URL install name/category guard"),
		hermesNestedSubcommand("skills", "tap", "list", "hermes_cli/main.py:tap_subparsers:list", "Skills hub direct URL install name/category guard"),
		hermesNestedSubcommand("skills", "tap", "add", "hermes_cli/main.py:tap_subparsers:add", "Skills hub direct URL install name/category guard"),
		hermesNestedSubcommand("skills", "tap", "remove", "hermes_cli/main.py:tap_subparsers:remove", "Skills hub direct URL install name/category guard"),
	)
	entries = append(entries, hermesNestedCommands("plugins", "hermes_cli/main.py:plugins_subparsers", "Plugin SDK", []string{"install", "update", "remove", "list", "enable", "disable"})...)
	entries = append(entries,
		hermesNestedAlias("plugins", "rm", "remove", "hermes_cli/main.py:plugins_subparsers:remove aliases", "Plugin SDK"),
		hermesNestedAlias("plugins", "uninstall", "remove", "hermes_cli/main.py:plugins_subparsers:remove aliases", "Plugin SDK"),
		hermesNestedAlias("plugins", "ls", "list", "hermes_cli/main.py:plugins_subparsers:list aliases", "Plugin SDK"),
	)
	entries = append(entries, hermesNestedCommands("memory", "hermes_cli/main.py:memory_sub", "Goncho memory integration into normal agent turn", []string{"setup", "status", "off", "reset"})...)
	entries = append(entries, hermesNestedCommands("tools", "hermes_cli/main.py:tools_sub", "Tool/runtime/security rows", []string{"list", "disable", "enable"})...)
	entries = append(entries, hermesNestedCommands("mcp", "hermes_cli/main.py:mcp_sub", "ACP server side", []string{"serve", "add", "remove", "list", "test", "configure", "login"})...)
	entries = append(entries,
		hermesNestedAlias("mcp", "rm", "remove", "hermes_cli/main.py:mcp_sub:remove aliases", "ACP server side"),
		hermesNestedAlias("mcp", "ls", "list", "hermes_cli/main.py:mcp_sub:list aliases", "ACP server side"),
		hermesNestedAlias("mcp", "config", "configure", "hermes_cli/main.py:mcp_sub:configure aliases", "ACP server side"),
	)
	entries = append(entries, hermesNestedCommands("sessions", "hermes_cli/main.py:sessions_subparsers", "Session shutdown memory transcript handoff", []string{"list", "export", "delete", "prune", "stats", "rename", "browse"})...)
	entries = append(entries, hermesKanbanCommands()...)
	entries = append(entries, hermesClawCommands()...)
	entries = append(entries, hermesCuratorCommands()...)
	entries = append(entries, hermesProfileCommands()...)
	entries = append(entries, hermesOwnedPath([]string{"agent", "reset"}, "cmd/gormes/agent.go:reset", "Gormes-owned default agent template reset command"))

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

func hermesOwnedPath(path []string, sourceRef, residual string) hermesCLIParityEntry {
	return hermesCLIParityEntry{Path: slices.Clone(path), Kind: hermesCLICommand, Status: hermesCLIOwned, SourceRef: sourceRef, Target: "cmd/gormes " + strings.Join(path, " "), Residual: residual}
}

func hermesImplementedPath(path []string, kind hermesCLIParityKind, sourceRef, target, residual string) hermesCLIParityEntry {
	entry := hermesCLIParityEntry{
		Path:      slices.Clone(path),
		Kind:      kind,
		Status:    hermesCLIImplemented,
		SourceRef: sourceRef,
		Target:    target,
		Residual:  residual,
	}
	markHermesCLIEntryFlags(&entry)
	return entry
}

func hermesExcludedCommand(name, sourceRef, residual string) hermesCLIParityEntry {
	return hermesCLIParityEntry{Path: []string{name}, Kind: hermesCLICommand, Status: hermesCLIExcluded, SourceRef: sourceRef, Residual: residual}
}

func hermesProviderLogoutCommand() hermesCLIParityEntry {
	entry := hermesRowCommand("logout", "hermes_cli/auth.py:logout_command", "Gormes top-level logout provider shortcut", "top-level provider logout remains row-backed; clears auth state and resets provider config")
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
		markHermesCLIEntryFlags(&entry)
		out = append(out, entry)
	}
	return out
}

func hermesFallbackCommands() []hermesCLIParityEntry {
	return []hermesCLIParityEntry{
		hermesImplementedPath([]string{"fallback", "list"}, hermesCLICommand, "hermes_cli/main.py:fallback_subparsers:list", "cmd/gormes fallback list", "lists the local fallback provider chain from Gormes config"),
		hermesImplementedPath([]string{"fallback", "ls"}, hermesCLIAlias, "hermes_cli/main.py:fallback_subparsers:list aliases", "cmd/gormes fallback list", "fallback ls alias resolves to fallback list"),
		hermesImplementedPath([]string{"fallback", "add"}, hermesCLICommand, "hermes_cli/fallback_cmd.py:cmd_fallback_add", "cmd/gormes fallback add", "appends a provider/model selected through the model picker without changing the primary model"),
		hermesImplementedPath([]string{"fallback", "remove"}, hermesCLICommand, "hermes_cli/fallback_cmd.py:cmd_fallback_remove", "cmd/gormes fallback remove", "removes a selected fallback entry from the local chain"),
		hermesImplementedPath([]string{"fallback", "rm"}, hermesCLIAlias, "hermes_cli/main.py:fallback_subparsers:remove aliases", "cmd/gormes fallback remove", "fallback rm alias resolves to fallback remove"),
		hermesImplementedPath([]string{"fallback", "clear"}, hermesCLICommand, "hermes_cli/fallback_cmd.py:cmd_fallback_clear", "cmd/gormes fallback clear", "clears the local fallback chain after confirmation"),
	}
}

func hermesClawCommands() []hermesCLIParityEntry {
	migrate := hermesCLIParityEntry{
		Path:      []string{"claw", "migrate"},
		Kind:      hermesCLICommand,
		Status:    hermesCLIImplemented,
		SourceRef: "hermes_cli/main.py:claw_subparsers:migrate",
		Target:    "cmd/gormes claw migrate",
		Row:       "OpenClaw migration dry-run manifest",
		Residual:  "`gormes claw migrate --dry-run` delegates to the Gormes-native OpenClaw migration manifest; full preview-then-prompt UX remains an owned CLI-safety divergence",
	}
	markHermesCLIEntryFlags(&migrate)

	cleanup := hermesRowPath([]string{"claw", "cleanup"}, hermesCLICommand, "hermes_cli/main.py:claw_subparsers:cleanup", "OpenClaw migration writer and cleanup command", "`gormes claw cleanup` delegates to the Gormes-native OpenClaw cleanup engine")
	cleanup.Status = hermesCLIImplemented
	cleanup.Target = "cmd/gormes claw cleanup"
	markHermesCLIEntryFlags(&cleanup)
	clean := hermesNestedAlias("claw", "clean", "cleanup", "hermes_cli/main.py:claw_subparsers:cleanup aliases", "OpenClaw migration writer and cleanup command")
	clean.Status = hermesCLIImplemented
	clean.Target = "cmd/gormes claw cleanup"
	return []hermesCLIParityEntry{migrate, cleanup, clean}
}

func hermesCuratorCommands() []hermesCLIParityEntry {
	const row = "Hermes curator archive/list/prune CLI catch-up"
	const source = "hermes_cli/curator.py:register_cli"
	commands := []string{"status", "run", "pause", "resume", "pin", "unpin", "restore", "list-archived", "archive", "prune", "backup", "rollback"}
	out := make([]hermesCLIParityEntry, 0, len(commands))
	for _, command := range commands {
		entry := hermesRowPath([]string{"curator", command}, hermesCLICommand, source+":"+command, row, "curator "+command+" is implemented over native Gormes curator state")
		entry.Status = hermesCLIImplemented
		entry.Target = "cmd/gormes curator " + command
		markHermesCLIEntryFlags(&entry)
		out = append(out, entry)
	}
	return out
}

func hermesProfileCommands() []hermesCLIParityEntry {
	const row = "Gormes profile command binding"
	const source = "hermes_cli/main.py:profile_subparsers"
	implemented := []struct {
		name     string
		target   string
		residual string
	}{
		{name: "list", target: "cmd/gormes profile list", residual: "profile list is implemented over native Gormes profile roots"},
		{name: "use", target: "cmd/gormes profile use", residual: "profile use is implemented as the canonical sticky active-profile switch; profile set remains a Gormes compatibility alias"},
		{name: "create", target: "cmd/gormes profile create", residual: "profile create is implemented for named Gormes profile roots with clone-all support"},
		{name: "show", target: "cmd/gormes profile show", residual: "profile show is implemented over the active Gormes profile with redacted root output"},
		{name: "info", target: "cmd/gormes profile info", residual: "profile info is implemented for distribution.yaml metadata"},
	}
	out := make([]hermesCLIParityEntry, 0, 12)
	for _, command := range implemented {
		entry := hermesImplementedPath([]string{"profile", command.name}, hermesCLICommand, source+":"+command.name, command.target, command.residual)
		out = append(out, entry)
	}
	for _, command := range []string{"delete", "alias", "rename", "export", "import", "install", "update"} {
		entry := hermesRowPath([]string{"profile", command}, hermesCLICommand, source+":"+command, row, "profile "+command+" is registered in Gormes as a deterministic row-backed unavailable command; full behavior remains row-backed")
		markHermesCLIEntryFlags(&entry)
		out = append(out, entry)
	}
	return out
}

func hermesGatewayNestedCommands() []hermesCLIParityEntry {
	out := hermesNestedCommands("gateway", "hermes_cli/main.py:gateway_subparsers", "Gateway, platform, webhook, and cron management CLI", []string{"run", "start", "stop", "restart", "status", "install", "uninstall", "setup", "migrate-legacy", "list"})
	for i := range out {
		key := strings.Join(out[i].Path, " ")
		switch key {
		case "gateway status":
			out[i].Status = hermesCLIImplemented
			out[i].Target = "cmd/gormes gateway status"
			out[i].Residual = "read-only gateway status command is implemented"
		case "gateway stop":
			out[i].Status = hermesCLIImplemented
			out[i].Target = "cmd/gormes gateway stop"
			out[i].Residual = "local gateway stop is implemented over validated runtime PID evidence"
		}
	}
	return out
}

func hermesKanbanCommands() []hermesCLIParityEntry {
	const row = "Hermes Kanban durable board core"
	const source = "hermes_cli/kanban.py:build_parser"
	core := []string{"init", "create", "list", "show", "claim", "complete", "block", "unblock", "promote", "link"}
	out := make([]hermesCLIParityEntry, 0, 32)
	for _, command := range core {
		entry := hermesRowPath([]string{"kanban", command}, hermesCLICommand, source+":"+command, row, "kanban "+command+" is implemented for the default local Gormes board")
		entry.Status = hermesCLIImplemented
		entry.Target = "cmd/gormes kanban " + command
		markHermesCLIEntryFlags(&entry)
		out = append(out, entry)
	}

	logEntry := hermesRowPath([]string{"kanban", "log"}, hermesCLICommand, source+":log", "Kanban worker log read command", "kanban log is implemented as a read-only selected-board worker log accessor")
	logEntry.Status = hermesCLIImplemented
	logEntry.Target = "cmd/gormes kanban log"
	out = append(out, logEntry)

	tailEntry := hermesRowPath([]string{"kanban", "tail"}, hermesCLICommand, source+":tail", "Kanban task event tail command", "kanban tail is implemented as a read-only selected-board task event follower")
	tailEntry.Status = hermesCLIImplemented
	tailEntry.Target = "cmd/gormes kanban tail"
	out = append(out, tailEntry)

	alias := hermesRowPath([]string{"kanban", "ls"}, hermesCLIAlias, source+":list aliases", row, "kanban ls alias remains row-backed until the CLI alias is wired")
	alias.AliasFor = []string{"kanban", "list"}
	out = append(out, alias)

	residual := []string{
		"boards",
		"assign",
		"unlink",
		"comment",
		"archive",
		"dispatch",
		"daemon",
		"watch",
		"stats",
		"notify-subscribe",
		"notify-list",
		"notify-unsubscribe",
		"runs",
		"heartbeat",
		"assignees",
		"context",
		"gc",
	}
	for _, command := range residual {
		entry := hermesRowPath([]string{"kanban", command}, hermesCLICommand, source+":"+command, row, "kanban "+command+" remains row-backed after the durable board core slice")
		markHermesCLIEntryFlags(&entry)
		out = append(out, entry)
	}
	return out
}

func hermesNestedAlias(group, alias, canonical, sourceRef, row string) hermesCLIParityEntry {
	entry := hermesRowPath([]string{group, alias}, hermesCLIAlias, sourceRef, row, group+" "+alias+" alias resolves to "+group+" "+canonical)
	entry.AliasFor = []string{group, canonical}
	markHermesCLIEntryFlags(&entry)
	return entry
}

func hermesNestedSubcommand(group, command, subcommand, sourceRef, row string) hermesCLIParityEntry {
	entry := hermesRowPath([]string{group, command, subcommand}, hermesCLICommand, sourceRef, row, group+" "+command+" "+subcommand+" handler remains classified by this manifest; implementation lands in its dedicated row")
	markHermesCLIEntryFlags(&entry)
	return entry
}

func markHermesCLIEntryFlags(entry *hermesCLIParityEntry) {
	key := strings.Join(entry.Path, " ")
	switch key {
	case "backup create",
		"claw cleanup", "claw clean",
		"cron remove", "cron rm", "cron delete",
		"curator archive", "curator prune", "curator rollback", "curator restore",
		"fallback remove", "fallback rm", "fallback clear",
		"hooks revoke", "hooks remove", "hooks rm",
		"kanban archive", "kanban gc",
		"mcp remove", "mcp rm",
		"pairing revoke", "pairing clear-pending",
		"plugins remove", "plugins rm", "plugins uninstall",
		"profile delete",
		"sessions delete", "sessions prune",
		"skills reset", "skills tap remove", "skills uninstall",
		"webhook remove", "webhook rm":
		entry.Destructive = true
	}
	switch key {
	case "config set", "auth login":
		entry.RedactsSecrets = true
	}
	switch key {
	case "claw migrate", "config migrate", "curator prune", "kanban promote":
		entry.DryRun = true
	}
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
		rowOverride string
	}{
		{
			name:        "add",
			sourceRef:   "hermes_cli/auth_commands.py:auth_add_command",
			residual:    "provider auth/add flow is implemented for API keys plus openai-codex, anthropic, nous, google-gemini-cli, and qwen-oauth native OAuth adapters; Spotify remains a separate service-provider subcommand",
			status:      hermesCLIImplemented,
			target:      "cmd/gormes auth add",
			redacts:     true,
			rowOverride: "Hermes auth OAuth provider adapters",
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
			name:        "spotify",
			sourceRef:   "hermes_cli/auth_commands.py:auth_spotify_command",
			residual:    "Spotify PKCE auth actions login|status|logout remain row-backed under provider auth CLI parity",
			redacts:     true,
			rowOverride: "Hermes auth Spotify service-provider subcommand",
		},
	}
	out := make([]hermesCLIParityEntry, 0, len(commands))
	for _, command := range commands {
		rowLabel := command.rowOverride
		if rowLabel == "" {
			rowLabel = "Hermes auth credential-pool command surface"
		}
		entry := hermesRowPath([]string{"auth", command.name}, hermesCLICommand, command.sourceRef, rowLabel, command.residual)
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
