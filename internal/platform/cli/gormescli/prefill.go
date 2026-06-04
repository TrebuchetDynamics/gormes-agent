package gormescli

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/app/prefill"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func ConfiguredPrefillMessages(cfg config.Config) []llm.Message {
	return prefill.ConfiguredMessages(cfg)
}
