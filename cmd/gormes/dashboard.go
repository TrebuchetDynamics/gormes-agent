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
	gatewaymodule "github.com/TrebuchetDynamics/gormes-agent/internal/cli/gormescli/modules/gateway"
)

func newDashboardCommand() *cobra.Command {
	return gatewaymodule.NewDashboardCommandWithSeams(gatewaymodule.DashboardCommandSeams{
		Run: runDashboardCommand,
	})
}

func runDashboardCommand(_ *cobra.Command, opts gatewaymodule.DashboardOptions) error {
	cfg := apiserver.Config{
		APIKey:             os.Getenv("GORMES_DASHBOARD_API_KEY"),
		DashboardBoundHost: "127.0.0.1",
		BuildInfo: apiserver.BuildInfo{
			Version:   Version,
			GitCommit: resolveGitCommit(),
			GitDirty:  resolveGitDirty(),
			GoVersion: runtime.Version(),
		},
	}
	if cfg.APIKey == "" {
		cfg.APIKey = "gormes-dashboard-dev"
	}

	srv, err := apiserver.NewServerChecked(cfg)
	if err != nil {
		return err
	}
	addr := fmt.Sprintf("127.0.0.1:%d", opts.Port)
	url := fmt.Sprintf("http://%s/dashboard", addr)
	fmt.Fprintf(os.Stderr, "Gormes dashboard starting at %s\n", url)
	if !opts.NoOpen {
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
