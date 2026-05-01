package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/apiserver"
)

func newDashboardCommand() *cobra.Command {
	var port int
	var noOpen bool

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
			url := fmt.Sprintf("http://%s/dashboard", addr)
			fmt.Fprintf(os.Stderr, "Gormes dashboard starting at %s\n", url)
			if !noOpen {
				go func() {
					time.Sleep(250 * time.Millisecond)
					if err := openDashboardURL(url); err != nil {
						fmt.Fprintf(os.Stderr, "Open %s in your browser (%v)\n", url, err)
					}
				}()
			}

			server := &http.Server{
				Addr:    addr,
				Handler: srv.Handler(),
			}
			return server.ListenAndServe()
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 43827, "Dashboard HTTP server port")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "do not open the dashboard in a browser")
	return cmd
}

func openDashboardURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
