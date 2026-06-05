package router

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	routerpkg "github.com/TrebuchetDynamics/gormes-agent/internal/provider/router"
)

type Options struct {
	LoadConfig func() (config.Config, error)
	LookupEnv  func(string) (string, bool)
}

type Request struct {
	DryRun bool
}

type ReadModel = routerpkg.ReadModel

func Run(_ context.Context, out io.Writer, request Request, opts Options) error {
	opts = normalizeOptions(opts)
	cfg, err := opts.LoadConfig()
	if err != nil {
		return fmt.Errorf("gormes router: load config: %w", err)
	}
	model := routerpkg.BuildReadModel(cfg, routerpkg.Options{LookupEnv: opts.LookupEnv})
	if request.DryRun {
		PrintDryRun(out, model)
		return nil
	}
	return fmt.Errorf("gormes router: HTTP serving is row-backed; use --dry-run to inspect config before the server-start slice")
}

func normalizeOptions(opts Options) Options {
	if opts.LoadConfig == nil {
		opts.LoadConfig = func() (config.Config, error) { return config.Load(nil) }
	}
	if opts.LookupEnv == nil {
		opts.LookupEnv = os.LookupEnv
	}
	return opts
}

func PrintDryRun(out io.Writer, model ReadModel) {
	fmt.Fprintln(out, "Gormes Router dry run")
	fmt.Fprintf(out, "enabled=%t\n", model.Enabled)
	fmt.Fprintf(out, "listen=%s\n", model.Listen)
	fmt.Fprintf(out, "openai_base_url=%s\n", OpenAIBaseURL(model.Listen))
	fmt.Fprintf(out, "state=%s\n", model.Status.State)
	fmt.Fprintf(out, "auth_configured=%t redacted=%t\n", model.Auth.Configured, model.Auth.Redacted)
	fmt.Fprintln(out, "dry_run_no_bind=true")
	registry := routerpkg.NewRegistry(model)
	for _, m := range registry.Models() {
		fmt.Fprintf(out, "model=%s provider=%s status=%s\n", m.ID, m.Provider, m.Status)
	}
}

func OpenAIBaseURL(listen string) string {
	listen = strings.TrimSpace(listen)
	if listen == "" {
		listen = routerpkg.DefaultListen
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
