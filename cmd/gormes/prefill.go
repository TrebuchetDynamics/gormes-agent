package main

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

func configuredPrefillMessages(cfg config.Config) []llm.Message {
	return gormescli.ConfiguredPrefillMessages(cfg)
}
