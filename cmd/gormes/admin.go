package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
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
			in, ok := cmd.InOrStdin().(*os.File)
			if !ok {
				fmt.Fprintln(cmd.ErrOrStderr(), "admin_tui_requires_tty: run `gormes admin` from an interactive terminal")
				return newExitCodeError(2, admin.ErrRequiresTTY)
			}
			screens := admin.NewDefaultScreens(
				admin.WithCommandEntries(adminCommandEntries(cmd.Root())),
				admin.WithCommandCatalogRunner(newAdminCommandRunner()),
			)
			err := admin.Run(in, cmd.OutOrStdout(), screens...)
			if errors.Is(err, admin.ErrRequiresTTY) {
				fmt.Fprintln(cmd.ErrOrStderr(), "admin_tui_requires_tty: run `gormes admin` from an interactive terminal")
				return newExitCodeError(2, err)
			}
			return err
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
	return func(entry admin.CommandEntry) admin.CommandRunResult {
		args, label, err := adminCommandRunArgs(entry.Path)
		result := admin.CommandRunResult{RunLabel: label}
		if err != nil {
			result.Error = err.Error()
			return result
		}

		root := newRootCommandWithRuntime(rootRuntime{})
		var stdout, stderr bytes.Buffer
		root.SetOut(&stdout)
		root.SetErr(&stderr)
		root.SetIn(strings.NewReader(""))
		err = executeRootCommand(root, args...)
		result.Output = strings.TrimRight(stdout.String()+stderr.String(), "\n")
		if err != nil {
			result.Error = err.Error()
			result.ExitCode = exitCodeFromError(err)
		}
		return result
	}
}

func adminCommandRunArgs(path string) ([]string, string, error) {
	return gormescli.AdminCommandRunArgs(path)
}
