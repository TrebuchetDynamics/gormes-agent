package skills

import "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/commands"

type SkillSlashCommand = commands.SkillSlashCommand

type SlashMessageOptions = commands.SlashMessageOptions

func BuildSkillSlashCommands(skills []Skill) []SkillSlashCommand {
	return commands.BuildSkillSlashCommands(skills)
}

func ResolveSkillSlashCommand(skillCommands []SkillSlashCommand, raw string) (SkillSlashCommand, bool) {
	return commands.ResolveSkillSlashCommand(skillCommands, raw)
}

func BuildSkillSlashCommandMessage(command SkillSlashCommand, userInstruction string, opts SlashMessageOptions) string {
	return commands.BuildSkillSlashCommandMessage(command, userInstruction, opts)
}

func normalizeSkillCommandName(name string) string {
	return commands.NormalizeSkillCommandName(name)
}
