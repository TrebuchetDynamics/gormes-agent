package main

import (
	"fmt"
	"log/slog"

	"github.com/TrebuchetDynamics/gormes-agent/internal/agenttemplate"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func ensureGatewayAgentTemplates(cfg config.Config, log *slog.Logger) (agenttemplate.WriteResult, error) {
	result, err := agenttemplate.ApplyDefaultTemplates(agenttemplate.WriteOptions{
		TargetDir: gatewayAgentTemplateTarget(cfg),
	})
	if err != nil {
		return result, fmt.Errorf("gateway agent templates: %w", err)
	}
	if log != nil {
		created, skipped, overwritten := countGatewayAgentTemplateActions(result)
		log.Info("gateway agent templates ready", "target", result.TargetDir, "created", created, "skipped", skipped, "overwritten", overwritten)
	}
	return result, nil
}

func gatewayAgentTemplateTarget(cfg config.Config) string {
	if target := gatewayContextFilesCWD(cfg); target != "" {
		return target
	}
	return config.GormesHome()
}

func countGatewayAgentTemplateActions(result agenttemplate.WriteResult) (created, skipped, overwritten int) {
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
