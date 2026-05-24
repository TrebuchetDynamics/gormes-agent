package main

import (
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli/gormescli"
)

func newDashboardCommand() *cobra.Command {
	return gormescli.NewDashboardCommand(gormescli.DashboardCommandOptions{
		APIKey:    os.Getenv("GORMES_DASHBOARD_API_KEY"),
		Version:   Version,
		GitCommit: resolveGitCommit(),
		GitDirty:  resolveGitDirty(),
		GoVersion: runtime.Version(),
		Stderr:    os.Stderr,
		OpenURL:   openDashboardURL,
	})
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
