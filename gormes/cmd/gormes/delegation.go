package main

import (
	"log/slog"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/subagent"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/tools"
)

func registerDelegation(cfg config.Config, reg *tools.Registry, hc hermes.Client) *subagent.Manager {
	if reg == nil || !cfg.Delegation.Enabled {
		return nil
	}
	if err := subagent.ValidateDelegateTimeout(cfg.Delegation.DefaultTimeout, "delegation default timeout"); err != nil {
		slog.Warn("delegate_task registration skipped: default timeout is outside the budget", "err", err, "default_timeout", cfg.Delegation.DefaultTimeout, "budget", 2*time.Minute-10*time.Second)
		return nil
	}

	runner := subagent.NewChatRunner(hc, reg, subagent.ChatRunnerConfig{
		Model:           cfg.Hermes.Model,
		MaxToolDuration: 30 * time.Second,
	})
	mgr := subagent.NewManager(cfg.Delegation, runner, cfg.DelegationRunLogPath())
	reg.MustRegister(subagent.NewDelegateTool(mgr))
	return mgr
}
