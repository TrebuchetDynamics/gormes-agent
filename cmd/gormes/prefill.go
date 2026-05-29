package main

import (
	"log/slog"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func configuredPrefillMessages(cfg config.Config) []llm.Message {
	messages, err := config.LoadConfiguredPrefillMessages(cfg)
	if err != nil {
		slog.Warn("prefill messages unavailable", "err", err)
		return nil
	}
	return messages
}
