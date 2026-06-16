package gateway

import (
	"log/slog"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/core/agenttemplate"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/gateway/agenttemplates"
)

func EnsureAgentTemplates(cfg config.Config, log *slog.Logger) (agenttemplate.WriteResult, error) {
	return agenttemplates.Ensure(cfg, log)
}

func AgentTemplateTarget(cfg config.Config) string {
	return agenttemplates.Target(cfg)
}

func CountAgentTemplateActions(result agenttemplate.WriteResult) (created, skipped, overwritten int) {
	return agenttemplates.CountActions(result)
}

func ContextFilesCWD(cfg config.Config) string {
	return agenttemplates.ContextFilesCWD(cfg)
}
