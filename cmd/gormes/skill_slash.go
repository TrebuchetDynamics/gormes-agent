package main

import (
	"context"
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func tuiSkillSlashCommands(ctx context.Context, cfg config.Config) []skills.SkillSlashCommand {
	commands, _, _ := loadTUISkillSlashCommands(ctx, cfg)
	return commands
}

func tuiSkillSlashReloadFunc(cfg config.Config) tui.SkillSlashReloadFunc {
	return func(ctx context.Context) (tui.SkillSlashReloadResult, error) {
		commands, _, err := loadTUISkillSlashCommands(ctx, cfg)
		if err != nil {
			return tui.SkillSlashReloadResult{}, err
		}
		return tui.SkillSlashReloadResult{
			Commands: commands,
			Output:   fmt.Sprintf("Skills Reloaded\n%d skill(s) available", len(commands)),
		}, nil
	}
}

func loadTUISkillSlashCommands(ctx context.Context, cfg config.Config) ([]skills.SkillSlashCommand, []skills.SkillStatus, error) {
	runtime := skills.NewRuntime(cfg.SkillsRoot(), cfg.Skills.MaxDocumentBytes, cfg.Skills.SelectionCap, cfg.SkillsUsageLogPath())
	return runtime.SkillSlashCommands(ctx, skills.RuntimeOptions{})
}
