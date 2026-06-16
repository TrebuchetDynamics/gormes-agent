package preferences

import (
	"context"
	"time"

	appusage "github.com/TrebuchetDynamics/gormes-agent/internal/app/usage"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func NewAccountUsageFunc(cfg config.Config) tui.AccountUsageFunc {
	return func(ctx context.Context) (llm.AccountUsageSnapshot, error) {
		resolution, _ := config.ResolveTUIInference(config.TUIInferenceRequest{Config: cfg, CommandLabel: "gormes TUI /usage"})
		provider := appusage.InferUsageProvider(resolution.Provider, appusage.FirstUsageString(resolution.Model, cfg.Hermes.Model))
		if provider == "" {
			provider = "openai-codex"
		}
		fetcher := llm.NewAccountUsageFetcher(appusage.AccountUsageHTTPClient{Client: appusage.UsageHTTPClient}, func() time.Time { return time.Now().UTC() })
		return fetcher.Fetch(ctx, llm.AccountUsageFetchRequest{
			Provider: provider,
			BaseURL:  cfg.Hermes.Endpoint,
			APIKey:   cfg.Hermes.APIKey,
		})
	}
}
