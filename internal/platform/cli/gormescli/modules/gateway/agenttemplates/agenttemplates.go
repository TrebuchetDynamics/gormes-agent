package agenttemplates

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/core/agenttemplate"
)

func Ensure(cfg config.Config, log *slog.Logger) (agenttemplate.WriteResult, error) {
	result, err := agenttemplate.ApplyDefaultTemplates(agenttemplate.WriteOptions{
		TargetDir: Target(cfg),
	})
	if err != nil {
		return result, fmt.Errorf("gateway agent templates: %w", err)
	}
	if log != nil {
		created, skipped, overwritten := CountActions(result)
		log.Info("gateway agent templates ready", "target", result.TargetDir, "created", created, "skipped", skipped, "overwritten", overwritten)
	}
	return result, nil
}

func Target(cfg config.Config) string {
	if target := ContextFilesCWD(cfg); target != "" {
		return target
	}
	return config.GormesHome()
}

func CountActions(result agenttemplate.WriteResult) (created, skipped, overwritten int) {
	for _, file := range result.Files {
		switch file.Action {
		case agenttemplate.ActionCreate:
			created++
		case agenttemplate.ActionSkip:
			skipped++
		case agenttemplate.ActionOverwrite:
			overwritten++
		}
	}
	return created, skipped, overwritten
}

func ContextFilesCWD(cfg config.Config) string {
	if cwd := strings.TrimSpace(cfg.Terminal.CWD); cwd != "" && cwd != "." {
		return cwd
	}
	if agent, ok := cfg.Agents.AgentByID(cfg.Agents.DefaultAgentID()); ok {
		return strings.TrimSpace(agent.Workspace)
	}
	return ""
}
