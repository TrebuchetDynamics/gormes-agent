package gormescli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/apiserver"
)

type DashboardCommandOptions struct {
	APIKey    string
	Version   string
	GitCommit string
	GitDirty  bool
	GoVersion string
	Stderr    io.Writer
	OpenURL   func(string) error
}

type dashboardOptions struct {
	Port   int
	NoOpen bool
}

func NewDashboardCommand(options DashboardCommandOptions) *cobra.Command {
	opts := dashboardOptions{}
	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Start the Gormes web dashboard",
		Long:  "Starts an HTTP server with an htmx-based web dashboard for managing sessions, config, skills, and logs.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runDashboardCommand(cmd, opts, options)
		},
	}
	cmd.Flags().IntVar(&opts.Port, "port", 43827, "Dashboard HTTP server port")
	cmd.Flags().BoolVar(&opts.NoOpen, "no-open", false, "do not open the dashboard in a browser")
	return cmd
}

func runDashboardCommand(_ *cobra.Command, opts dashboardOptions, options DashboardCommandOptions) error {
	apiKey := options.APIKey
	if apiKey == "" {
		apiKey = "gormes-dashboard-dev"
	}
	cfg := apiserver.Config{
		APIKey:             apiKey,
		DashboardBoundHost: "127.0.0.1",
		BuildInfo: apiserver.BuildInfo{
			Version:   options.Version,
			GitCommit: options.GitCommit,
			GitDirty:  options.GitDirty,
			GoVersion: options.GoVersion,
		},
	}

	srv, err := apiserver.NewServerChecked(cfg)
	if err != nil {
		return err
	}
	addr := fmt.Sprintf("127.0.0.1:%d", opts.Port)
	url := fmt.Sprintf("http://%s/dashboard", addr)
	stderr := options.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	fmt.Fprintf(stderr, "Gormes dashboard starting at %s\n", url)
	if !opts.NoOpen {
		openURL := options.OpenURL
		go func() {
			time.Sleep(250 * time.Millisecond)
			if openURL == nil {
				fmt.Fprintf(stderr, "Open %s in your browser (no opener configured)\n", url)
				return
			}
			if err := openURL(url); err != nil {
				fmt.Fprintf(stderr, "Open %s in your browser (%v)\n", url, err)
			}
		}()
	}

	server := &http.Server{
		Addr:    addr,
		Handler: srv.Handler(),
	}
	return server.ListenAndServe()
}
