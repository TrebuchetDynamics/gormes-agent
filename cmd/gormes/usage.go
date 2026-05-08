package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/spf13/cobra"
)

func newUsageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "usage",
		Short: "Show runtime/provider account usage",
		RunE: func(cmd *cobra.Command, args []string) error {
			provider, _ := cmd.Flags().GetString("provider")
			apiKey, _ := cmd.Flags().GetString("api-key")
			baseURL, _ := cmd.Flags().GetString("base-url")
			accountID, _ := cmd.Flags().GetString("account-id")
			return runUsageCommand(cmd, usageInvocation{Provider: provider, APIKey: apiKey, BaseURL: baseURL, AccountID: accountID})
		},
	}
	cmd.Flags().String("provider", "", "provider account usage to query (openai-codex, anthropic, openrouter)")
	cmd.Flags().String("api-key", "", "provider API/OAuth token for account usage; defaults to configured hermes api_key")
	cmd.Flags().String("base-url", "", "provider account usage base URL override")
	cmd.Flags().String("account-id", "", "provider account identifier when required")
	return cmd
}

type usageInvocation struct {
	Provider  string
	APIKey    string
	BaseURL   string
	AccountID string
}

// usageHTTPClient bounds the provider account-usage fetch so an
// unresponsive provider can't hang the operator's terminal. 30s gives
// slow providers room to respond while preventing indefinite hangs;
// http.DefaultClient has no timeout at all.
var usageHTTPClient = &http.Client{Timeout: 30 * time.Second}

func runUsageCommand(cmd *cobra.Command, invocation usageInvocation) error {
	cfg, err := config.Load(nil)
	if err != nil {
		return err
	}
	provider := strings.TrimSpace(invocation.Provider)
	if provider == "" {
		resolution, _ := config.ResolveTUIInference(config.TUIInferenceRequest{Config: cfg, CommandLabel: "gormes usage"})
		provider = inferUsageProvider(resolution.Provider, firstUsageString(resolution.Model, cfg.Hermes.Model))
	}
	key := firstUsageString(invocation.APIKey, cfg.Hermes.APIKey)
	baseURL := firstUsageString(invocation.BaseURL, cfg.Hermes.Endpoint)
	fetcher := hermes.NewAccountUsageFetcher(accountUsageHTTPClient{client: usageHTTPClient}, func() time.Time { return time.Now().UTC() })
	snapshot, err := fetcher.Fetch(cmd.Context(), hermes.AccountUsageFetchRequest{
		Provider:  provider,
		BaseURL:   baseURL,
		APIKey:    key,
		AccountID: invocation.AccountID,
	})
	if err != nil {
		return err
	}
	for _, line := range hermes.RenderAccountUsageLines(snapshot, hermes.AccountUsageRenderOptions{}) {
		fmt.Fprintln(cmd.OutOrStdout(), line)
	}
	return nil
}

type accountUsageHTTPClient struct{ client *http.Client }

func (c accountUsageHTTPClient) DoAccountUsageRequest(ctx context.Context, req hermes.AccountUsageHTTPRequest) (hermes.AccountUsageHTTPResponse, error) {
	client := c.client
	if client == nil {
		client = http.DefaultClient
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, req.URL, nil)
	if err != nil {
		return hermes.AccountUsageHTTPResponse{}, err
	}
	for key, value := range req.Headers {
		if strings.TrimSpace(value) != "" {
			httpReq.Header.Set(key, value)
		}
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return hermes.AccountUsageHTTPResponse{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return hermes.AccountUsageHTTPResponse{}, err
	}
	return hermes.AccountUsageHTTPResponse{StatusCode: resp.StatusCode, Body: body}, nil
}

func inferUsageProvider(configuredProvider, model string) string {
	provider := strings.TrimSpace(configuredProvider)
	if provider != "" {
		return provider
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return ""
	}
	for _, candidate := range []string{"openai-codex", "anthropic", "openai", "openrouter"} {
		if metadata := hermes.LookupModelMetadata(hermes.ModelRegistryQuery{Provider: candidate, Model: model}); metadata.Found {
			return metadata.Provider
		}
	}
	lower := strings.ToLower(model)
	if strings.HasPrefix(lower, "gpt-") || strings.HasPrefix(lower, "o1") || strings.HasPrefix(lower, "o3") || strings.HasPrefix(lower, "o4") {
		return "openai-codex"
	}
	if strings.Contains(lower, "claude") {
		return "anthropic"
	}
	return ""
}

func firstUsageString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
