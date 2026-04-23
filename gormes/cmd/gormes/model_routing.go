package main

import (
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/kernel"
)

func smartModelRouting(cfg config.Config) kernel.SmartModelRouting {
	return kernel.SmartModelRouting{
		Enabled:        cfg.Hermes.SmartRouting.Enabled,
		SimpleModel:    cfg.Hermes.SmartRouting.SimpleModel,
		MaxSimpleChars: cfg.Hermes.SmartRouting.MaxSimpleChars,
		MaxSimpleWords: cfg.Hermes.SmartRouting.MaxSimpleWords,
	}
}
