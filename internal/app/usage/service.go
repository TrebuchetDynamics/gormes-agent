package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/providers/usagehttp"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/providers/usagepolicy"
	"github.com/spf13/cobra"
)

// UsageSeams isolates config loading and provider account usage fetching for
// hermetic command tests.
type UsageSeams struct {
	LoadConfig        func() (config.Config, error)
	FetchAccountUsage func(context.Context, llm.AccountUsageFetchRequest) (llm.AccountUsageSnapshot, error)
}

// DefaultUsageSeams returns the production usage-command seams.
func DefaultUsageSeams() UsageSeams {
	return UsageSeams{
		LoadConfig: func() (config.Config, error) {
			return config.Load(nil)
		},
		FetchAccountUsage: func(ctx context.Context, req llm.AccountUsageFetchRequest) (llm.AccountUsageSnapshot, error) {
			fetcher := llm.NewAccountUsageFetcher(AccountUsageHTTPClient{Client: UsageHTTPClient}, func() time.Time { return time.Now().UTC() })
			return fetcher.Fetch(ctx, req)
		},
	}
}

func (s UsageSeams) withDefaults() UsageSeams {
	defaults := DefaultUsageSeams()
	if s.LoadConfig == nil {
		s.LoadConfig = defaults.LoadConfig
	}
	if s.FetchAccountUsage == nil {
		s.FetchAccountUsage = defaults.FetchAccountUsage
	}
	return s
}

// UsageReportJSON is the wire shape for `gormes usage --json`.
//
// Fleet automation tracking provider account usage across machines parses this
// to plot dashboards. Build provenance leads, matching the rest of the JSON
// command arc. The snapshot embeds AccountUsageSnapshot, which already has JSON
// tags.
type BuildProvenance struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

// Options carries binary-owned values into importable usage command code without
// making it depend on cmd/gormes.
type Options struct {
	BuildProvenance func() BuildProvenance
}

func (o Options) buildProvenance() BuildProvenance {
	if o.BuildProvenance == nil {
		return BuildProvenance{}
	}
	return o.BuildProvenance()
}

type UsageReportJSON struct {
	Build BuildProvenance `json:"build"`
	llm.AccountUsageSnapshot
}

// NewUsageCommand creates the provider account-usage command.
func NewUsageCommand(opts Options) *cobra.Command {
	return NewUsageCommandWithSeams(UsageSeams{}, opts)
}

// NewUsageCommandWithSeams creates the provider account-usage command with
// injectable runtime seams.
func NewUsageCommandWithSeams(seams UsageSeams, opts Options) *cobra.Command {
	seams = seams.withDefaults()
	cmd := &cobra.Command{
		Use:   "usage",
		Short: "Show runtime/provider account usage",
		RunE: func(cmd *cobra.Command, args []string) error {
			provider, _ := cmd.Flags().GetString("provider")
			apiKey, _ := cmd.Flags().GetString("api-key")
			baseURL, _ := cmd.Flags().GetString("base-url")
			accountID, _ := cmd.Flags().GetString("account-id")
			return runUsageCommand(cmd, UsageInvocation{Provider: provider, APIKey: apiKey, BaseURL: baseURL, AccountID: accountID}, seams, opts)
		},
	}
	cmd.Flags().String("provider", "", "provider account usage to query (openai-codex, anthropic, openrouter)")
	cmd.Flags().String("api-key", "", "provider API/OAuth token for account usage; defaults to configured hermes api_key")
	cmd.Flags().String("base-url", "", "provider account usage base URL override")
	cmd.Flags().String("account-id", "", "provider account identifier when required")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: {build, provider, account_id, plan, source, fetched_at, windows: [...], details, unavailable}")
	return cmd
}

// UsageInvocation is the parsed flag payload for a usage command run.
type UsageInvocation struct {
	Provider  string
	APIKey    string
	BaseURL   string
	AccountID string
}

// UsageHTTPClient bounds the provider account-usage fetch so an unresponsive
// provider cannot hang the operator's terminal. http.DefaultClient has no
// timeout.
var UsageHTTPClient = usagehttp.DefaultClient

// RunUsageCommand executes the usage command with production seams.
func RunUsageCommand(cmd *cobra.Command, invocation UsageInvocation, opts Options) error {
	return runUsageCommand(cmd, invocation, UsageSeams{}, opts)
}

func runUsageCommand(cmd *cobra.Command, invocation UsageInvocation, seams UsageSeams, opts Options) error {
	seams = seams.withDefaults()
	cfg, err := seams.LoadConfig()
	if err != nil {
		return err
	}
	provider := strings.TrimSpace(invocation.Provider)
	if provider == "" {
		resolution, _ := config.ResolveTUIInference(config.TUIInferenceRequest{Config: cfg, CommandLabel: "gormes usage"})
		provider = InferUsageProvider(resolution.Provider, FirstUsageString(resolution.Model, cfg.Hermes.Model))
	}
	key := FirstUsageString(invocation.APIKey, cfg.Hermes.APIKey)
	baseURL := FirstUsageString(invocation.BaseURL, cfg.Hermes.Endpoint)
	snapshot, err := seams.FetchAccountUsage(cmd.Context(), llm.AccountUsageFetchRequest{
		Provider:  provider,
		BaseURL:   baseURL,
		APIKey:    key,
		AccountID: invocation.AccountID,
	})
	if err != nil {
		return err
	}
	asJSON, _ := cmd.Flags().GetBool("json")
	if asJSON {
		body, marshalErr := json.MarshalIndent(UsageReportJSON{
			Build:                opts.buildProvenance(),
			AccountUsageSnapshot: snapshot,
		}, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(body))
		return nil
	}
	for _, line := range llm.RenderAccountUsageLines(snapshot, llm.AccountUsageRenderOptions{}) {
		fmt.Fprintln(cmd.OutOrStdout(), line)
	}
	return nil
}

// AccountUsageHTTPClient adapts net/http to llm.AccountUsageHTTPClient.
type AccountUsageHTTPClient = usagehttp.Client

// InferUsageProvider infers the provider from configured provider/model
// settings when `gormes usage --provider` is not passed.
func InferUsageProvider(configuredProvider, model string) string {
	return usagepolicy.InferProvider(configuredProvider, model)
}

// FirstUsageString returns the first non-blank value.
func FirstUsageString(values ...string) string {
	return usagepolicy.FirstNonBlank(values...)
}
