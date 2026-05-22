package gormescli

import "github.com/TrebuchetDynamics/gormes-agent/internal/progress"

// DefaultContracts returns the first importable ownership map for the current
// gormes CLI surface. The declarations intentionally own command roots with
// descendants instead of mirroring every subcommand by hand; tests derive the
// exact manifest from the live Cobra tree.
func DefaultContracts() []ModuleContract {
	return []ModuleContract{
		{
			Module:        progress.ModuleBrowser,
			Commands:      commands("browser"),
			SlashCommands: slashCommands("browser"),
		},
		{
			Module: progress.ModuleChannels,
			Commands: commands(
				"channels",
				"slack",
				"telegram",
				"whatsapp",
			),
		},
		{
			Module: progress.ModuleCLI,
			Commands: commands(
				"backup",
				"completion",
				"debug",
				"dump",
				"import",
				"logs",
				"send",
				"setup",
				"version",
			),
			SlashCommands: slashCommands(
				"clear",
				"commands",
				"debug",
				"help",
				"quit",
				"redraw",
			),
		},
		{
			Module: progress.ModuleConfig,
			Commands: commands(
				"claw",
				"config",
				"migrate",
				"secrets",
			),
			SlashCommands: slashCommands("config"),
		},
		{
			Module:   progress.ModuleDoctor,
			Commands: commands("doctor"),
		},
		{
			Module:   progress.ModuleDocs,
			Commands: commands("fidelity"),
		},
		{
			Module:        progress.ModuleFleet,
			Commands:      commands("cron"),
			SlashCommands: slashCommands("background", "cron", "goal"),
		},
		{
			Module: progress.ModuleGateway,
			Commands: commands(
				"agent",
				"dashboard",
				"gateway",
				"hooks",
				"pairing",
				"webhook",
			),
			SetupSections: setupSections("agent", "bindings", "gateway", "workspace"),
			SlashCommands: slashCommands(
				"approve",
				"deny",
				"footer",
				"platform",
				"platforms",
				"sethome",
				"spawn",
				"topic",
				"verbose",
			),
		},
		{
			Module:   progress.ModuleGoncho,
			Commands: commands("goncho"),
		},
		{
			Module:        progress.ModuleInstall,
			Commands:      commands("restore", "uninstall", "update"),
			SlashCommands: slashCommands("update"),
		},
		{
			Module:        progress.ModuleKanban,
			Commands:      commands("kanban"),
			SlashCommands: slashCommands("kanban"),
		},
		{
			Module:   progress.ModuleMemory,
			Commands: commands("memory"),
		},
		{
			Module:        progress.ModuleNavivox,
			Commands:      commands("navivox"),
			SetupSections: setupSections("navivox"),
		},
		{
			Module: progress.ModuleProfiles,
			Commands: commands(
				"profile",
			),
			SetupSections: setupSections("profiles"),
			SlashCommands: slashCommands(
				"agents",
				"personality",
				"profile",
			),
		},
		{
			Module: progress.ModuleProviders,
			Commands: commands(
				"auth",
				"fallback",
				"insights",
				"logout",
				"model",
				"usage",
			),
			SetupSections: setupSections("model", "provider", "fallback"),
			SlashCommands: slashCommands(
				"gquota",
				"insights",
				"model",
				"usage",
			),
		},
		{
			Module: progress.ModuleRuntime,
			Commands: commands(
				"security",
				"status",
				"system",
			),
			SlashCommands: slashCommands(
				"busy",
				"fast",
				"new",
				"reasoning",
				"reload",
				"reload-mcp",
				"restart",
				"status",
				"stop",
			),
		},
		{
			Module: progress.ModuleSessions,
			Commands: commands(
				"checkpoints",
				"session",
			),
			SlashCommands: slashCommands(
				"branch",
				"compress",
				"history",
				"queue",
				"resume",
				"retry",
				"rollback",
				"save",
				"sessions",
				"snapshot",
				"steer",
				"title",
				"undo",
			),
		},
		{
			Module: progress.ModuleSkills,
			Commands: commands(
				"curator",
				"plugins",
				"skills",
			),
			SlashCommands: slashCommands(
				"curator",
				"plugins",
				"skills",
			),
		},
		{
			Module: progress.ModuleTools,
			Commands: commands(
				"acp",
				"mcp",
				"tools",
			),
			SetupSections: setupSections("tools"),
			SlashCommands: slashCommands(
				"tools",
				"toolsets",
				"yolo",
			),
		},
		{
			Module:        progress.ModuleTTS,
			SetupSections: setupSections("tts"),
			SlashCommands: slashCommands("tts", "voice"),
		},
		{
			Module: progress.ModuleTUI,
			Commands: commands(
				"admin",
				"chat",
			),
			SetupSections: setupSections("terminal"),
			SlashCommands: slashCommands(
				"compact",
				"copy",
				"details",
				"image",
				"indicator",
				"paste",
				"skin",
				"statusbar",
			),
		},
	}
}

func commands(paths ...string) []CommandSpec {
	out := make([]CommandSpec, 0, len(paths))
	for _, path := range paths {
		out = append(out, CommandSpec{Path: path, IncludeDescendants: true})
	}
	return out
}

func setupSections(names ...string) []SetupSectionSpec {
	out := make([]SetupSectionSpec, 0, len(names))
	for _, name := range names {
		out = append(out, SetupSectionSpec{Name: name})
	}
	return out
}

func slashCommands(names ...string) []SlashCommandSpec {
	out := make([]SlashCommandSpec, 0, len(names))
	for _, name := range names {
		out = append(out, SlashCommandSpec{Name: name})
	}
	return out
}
