package main

import (
	"bytes"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/admin"
)

func newAdminCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "admin",
		Short:        "Open the unified admin TUI",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return gormescli.AdminRun(cmd, gormescli.AdminOptions{Runner: newAdminCommandRunner()})
		},
	}
	return cmd
}

func adminCommandEntries(root *cobra.Command) []admin.CommandEntry {
	return gormescli.AdminCommandEntries(root)
}

func adminCommandRunnable(path string) bool {
	return gormescli.AdminCommandRunnable(path)
}

func adminCommandRunLabel(path string) string {
	return gormescli.AdminCommandRunLabel(path)
}

func newAdminCommandRunner() admin.CommandRunner {
	return gormescli.AdminCommandRunner(gormescli.AdminRunnerOptions{
		ExecuteCommand: func(args []string) (string, string, error) {
			root := newRootCommandWithRuntime(rootRuntime{})
			var stdout, stderr bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(&stderr)
			root.SetIn(strings.NewReader(""))
			err := executeRootCommand(root, args...)
			return stdout.String(), stderr.String(), err
		},
		ExitCode: exitCodeFromError,
	})
}

func adminCommandRunArgs(path string) ([]string, string, error) {
	return gormescli.AdminCommandRunArgs(path)
}
