package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/apiserver"
)

func newDashboardCommand() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Start the Gormes web dashboard",
		Long:  "Starts an HTTP server with an htmx-based web dashboard for managing sessions, config, skills, and logs.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg := apiserver.Config{
				APIKey: os.Getenv("GORMES_DASHBOARD_API_KEY"),
			}
			if cfg.APIKey == "" {
				cfg.APIKey = "gormes-dashboard-dev"
			}

			srv := apiserver.NewServer(cfg)
			addr := fmt.Sprintf("127.0.0.1:%d", port)
			fmt.Fprintf(os.Stderr, "Gormes dashboard starting at http://%s/dashboard\n", addr)

			server := &http.Server{
				Addr:    addr,
				Handler: srv.Handler(),
			}
			return server.ListenAndServe()
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 43827, "Dashboard HTTP server port")
	return cmd
}
