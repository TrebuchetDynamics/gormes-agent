package main

import (
	"context"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func newTUIAccountUsageFunc(cfg config.Config) tui.AccountUsageFunc {
	return func(ctx context.Context) (hermes.AccountUsageSnapshot, error) {
		resolution, _ := config.ResolveTUIInference(config.TUIInferenceRequest{Config: cfg, CommandLabel: "gormes TUI /usage"})
		provider := inferUsageProvider(resolution.Provider, firstUsageString(resolution.Model, cfg.Hermes.Model))
		if provider == "" {
			provider = "openai-codex"
		}
		fetcher := hermes.NewAccountUsageFetcher(accountUsageHTTPClient{client: usageHTTPClient}, func() time.Time { return time.Now().UTC() })
		return fetcher.Fetch(ctx, hermes.AccountUsageFetchRequest{
			Provider: provider,
			BaseURL:  cfg.Hermes.Endpoint,
			APIKey:   cfg.Hermes.APIKey,
		})
	}
}
