package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
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
				Path:     path,
				Use:      use,
				Short:    child.Short,
				Runnable: adminCommandRunnable(path),
				RunLabel: adminCommandRunLabel(path),
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

func adminCommandRunnable(path string) bool {
	switch path {
	case "doctor", "auth status", "gateway status", "kanban list":
		return true
	default:
		return false
	}
}

func adminCommandRunLabel(path string) string {
	switch path {
	case "doctor":
		return "gormes doctor --offline"
	case "auth status":
		return "gormes auth status <configured-provider>"
	case "gateway status":
		return "gormes gateway status"
	case "kanban list":
		return "gormes kanban list"
	default:
		return ""
	}
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
	switch path {
	case "doctor":
		return []string{"doctor", "--offline"}, "gormes doctor --offline", nil
	case "auth status":
		cfg, err := config.Load(nil)
		if err != nil {
			return nil, "gormes auth status <configured-provider>", fmt.Errorf("load config: %w", err)
		}
		provider := strings.TrimSpace(cfg.Hermes.Provider)
		if provider == "" {
			return nil, "gormes auth status <configured-provider>", fmt.Errorf("no configured provider for auth status")
		}
		return []string{"auth", "status", provider}, "gormes auth status " + provider, nil
	case "gateway status":
		return []string{"gateway", "status"}, "gormes gateway status", nil
	case "kanban list":
		return []string{"kanban", "list"}, "gormes kanban list", nil
	default:
		return nil, "gormes " + path, fmt.Errorf("command is not runnable inside gormes admin")
	}
}
