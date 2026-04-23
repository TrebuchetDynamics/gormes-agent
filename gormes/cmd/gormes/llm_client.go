package main

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
)

func newLLMClient(cfg config.Config) (hermes.Client, string) {
	endpoint := hermes.EffectiveEndpoint(cfg.Hermes.Provider, cfg.Hermes.Endpoint)
	return hermes.NewClient(cfg.Hermes.Provider, endpoint, cfg.Hermes.APIKey), endpoint
}

func llmProviderLabel(provider string) string {
	if strings.EqualFold(strings.TrimSpace(provider), "anthropic") {
		return "anthropic"
	}
	if strings.EqualFold(strings.TrimSpace(provider), "codex") || strings.EqualFold(strings.TrimSpace(provider), "openai-codex") {
		return "codex"
	}
	return "api_server"
}
