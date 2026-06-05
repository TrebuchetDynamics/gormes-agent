package gormescli

import (
	"github.com/spf13/cobra"

	appadmin "github.com/TrebuchetDynamics/gormes-agent/internal/app/admin"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	tuiadmin "github.com/TrebuchetDynamics/gormes-agent/internal/tui/admin"
)

type AdminOptions = appadmin.Options
type AdminRunnerOptions = appadmin.RunnerOptions

type AdminCommandOptions struct {
	Runner tuiadmin.CommandRunner
}

func NewAdminCommand(opts AdminCommandOptions) *cobra.Command {
	return &cobra.Command{
		Use:          "admin",
		Short:        "Open the unified admin TUI",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return AdminRun(cmd, AdminOptions{Runner: opts.Runner})
		},
	}
}

func AdminRun(cmd *cobra.Command, opts AdminOptions) error {
	return appadmin.Run(cmd, opts)
}

func AdminCommandEntries(root *cobra.Command) []tuiadmin.CommandEntry {
	return appadmin.CommandEntries(root)
}

func AdminCommandRunnable(path string) bool {
	return appadmin.CommandRunnable(path)
}

func AdminCommandRunLabel(path string) string {
	return appadmin.CommandRunLabel(path)
}

func AdminCommandRunArgs(path string) ([]string, string, error) {
	return appadmin.CommandRunArgs(path, func() (config.Config, error) { return config.Load(nil) })
}

func AdminCommandRunner(opts AdminRunnerOptions) tuiadmin.CommandRunner {
	return appadmin.CommandRunner(opts)
}
