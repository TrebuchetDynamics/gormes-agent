package main

import (
	"context"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	providermodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/providers"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func newTUIAccountUsageFunc(cfg config.Config) tui.AccountUsageFunc {
	return func(ctx context.Context) (llm.AccountUsageSnapshot, error) {
		resolution, _ := config.ResolveTUIInference(config.TUIInferenceRequest{Config: cfg, CommandLabel: "gormes TUI /usage"})
		provider := providermodule.InferUsageProvider(resolution.Provider, providermodule.FirstUsageString(resolution.Model, cfg.Hermes.Model))
		if provider == "" {
			provider = "openai-codex"
		}
		fetcher := llm.NewAccountUsageFetcher(providermodule.AccountUsageHTTPClient{Client: providermodule.UsageHTTPClient}, func() time.Time { return time.Now().UTC() })
		return fetcher.Fetch(ctx, llm.AccountUsageFetchRequest{
			Provider: provider,
			BaseURL:  cfg.Hermes.Endpoint,
			APIKey:   cfg.Hermes.APIKey,
		})
	}
}
