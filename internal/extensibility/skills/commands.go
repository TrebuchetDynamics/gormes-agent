package skills

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/commands"
)

type SkillSlashCommand = commands.SkillSlashCommand

type SlashMessageOptions = commands.SlashMessageOptions

func (r *Runtime) SkillSlashCommands(ctx context.Context, opts RuntimeOptions) ([]SkillSlashCommand, []SkillStatus, error) {
	if r == nil || r.store == nil {
		return nil, nil, nil
	}
	snapshot, err := r.store.SnapshotActive()
	if err != nil {
		return nil, nil, err
	}
	prepared, statuses := prepareSkills(ctx, snapshot.Skills, opts)
	return commands.BuildSkillSlashCommands(prepared), statuses, nil
}

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
