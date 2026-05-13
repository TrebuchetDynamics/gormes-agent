package main

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

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
			screens := admin.NewDefaultScreens(admin.WithCommandEntries(adminCommandEntries(cmd.Root())))
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
	if root == nil {
		return nil
	}
	rootPath := strings.TrimSpace(root.CommandPath())
	var entries []admin.CommandEntry
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		for _, child := range cmd.Commands() {
			if child.Hidden || child.Name() == "help" {
				continue
			}
			path := strings.TrimSpace(child.CommandPath())
			if rootPath != "" {
				path = strings.TrimSpace(strings.TrimPrefix(path, rootPath))
			}
			use := strings.TrimSpace(child.UseLine())
			if rootPath != "" {
				use = strings.TrimSpace(strings.TrimPrefix(use, rootPath))
			}
			if use != "" {
				use = "gormes " + use
			} else if path != "" {
				use = "gormes " + path
			}
			entries = append(entries, admin.CommandEntry{
				Path:  path,
				Use:   use,
				Short: child.Short,
			})
			walk(child)
		}
	}
	walk(root)
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})
	return entries
}
