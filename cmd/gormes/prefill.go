package main

import (
	prefillcmd "github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/prefillcmd"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func configuredPrefillMessages(cfg config.Config) []llm.Message {
	return prefillcmd.ConfiguredMessages(cfg)
}
