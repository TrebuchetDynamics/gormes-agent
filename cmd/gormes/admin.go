package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

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
			err := admin.Run(in, cmd.OutOrStdout(), admin.NewDefaultScreens()...)
			if errors.Is(err, admin.ErrRequiresTTY) {
				fmt.Fprintln(cmd.ErrOrStderr(), "admin_tui_requires_tty: run `gormes admin` from an interactive terminal")
				return newExitCodeError(2, err)
			}
			return err
		},
	}
	return cmd
}
