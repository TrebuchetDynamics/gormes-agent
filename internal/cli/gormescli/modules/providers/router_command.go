package providers

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	routerpkg "github.com/TrebuchetDynamics/gormes-agent/internal/provider/router"
	"github.com/spf13/cobra"
)

type RouterCommandOptions struct {
	LoadConfig func() (config.Config, error)
	LookupEnv  func(string) (string, bool)
}

func NewRouterCommand(opts RouterCommandOptions) *cobra.Command {
	if opts.LoadConfig == nil {
		opts.LoadConfig = func() (config.Config, error) { return config.Load(nil) }
	}
	if opts.LookupEnv == nil {
		opts.LookupEnv = os.LookupEnv
	}
	var dryRun bool
	cmd := &cobra.Command{
		Use:          "router",
		Short:        "Inspect or run the local OpenAI-compatible Gormes Router",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := opts.LoadConfig()
			if err != nil {
				return fmt.Errorf("gormes router: load config: %w", err)
			}
			model := routerpkg.BuildReadModel(cfg, routerpkg.Options{LookupEnv: opts.LookupEnv})
			if dryRun {
				printRouterDryRun(cmd.OutOrStdout(), model)
				return nil
			}
			return fmt.Errorf("gormes router: HTTP serving is row-backed; use --dry-run to inspect config before the server-start slice")
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "render Router listen/status/client config without binding a port")
	return cmd
}

func printRouterDryRun(out io.Writer, model routerpkg.ReadModel) {
	fmt.Fprintln(out, "Gormes Router dry run")
	fmt.Fprintf(out, "enabled=%t\n", model.Enabled)
	fmt.Fprintf(out, "listen=%s\n", model.Listen)
	fmt.Fprintf(out, "openai_base_url=%s\n", routerOpenAIBaseURL(model.Listen))
	fmt.Fprintf(out, "state=%s\n", model.Status.State)
	fmt.Fprintf(out, "auth_configured=%t redacted=%t\n", model.Auth.Configured, model.Auth.Redacted)
	fmt.Fprintln(out, "dry_run_no_bind=true")
	registry := routerpkg.NewRegistry(model)
	for _, m := range registry.Models() {
		fmt.Fprintf(out, "model=%s provider=%s status=%s\n", m.ID, m.Provider, m.Status)
	}
}

func routerOpenAIBaseURL(listen string) string {
	listen = strings.TrimSpace(listen)
	if listen == "" {
		listen = RouterDefaultListen
	}
	if !strings.Contains(listen, "://") {
		listen = "http://" + listen
	}
	parsed, err := url.Parse(listen)
	if err != nil || parsed.Host == "" {
		return strings.TrimRight(listen, "/") + "/v1"
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/v1"
	return parsed.String()
}
