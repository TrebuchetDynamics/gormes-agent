package admin

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	tuiadmin "github.com/TrebuchetDynamics/gormes-agent/internal/tui/admin"
)

type ConfigLoader func() (config.Config, error)

func CommandEntries(root *cobra.Command) []tuiadmin.CommandEntry {
	if root == nil {
		return nil
	}
	rootPath := strings.TrimSpace(root.CommandPath())
	var entries []tuiadmin.CommandEntry
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
			entries = append(entries, tuiadmin.CommandEntry{
				Path:     path,
				Use:      use,
				Short:    child.Short,
				Runnable: CommandRunnable(path),
				RunLabel: CommandRunLabel(path),
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

func CommandRunnable(path string) bool {
	switch path {
	case "doctor", "auth status", "gateway status", "kanban list":
		return true
	default:
		return false
	}
}

func CommandRunLabel(path string) string {
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

func CommandRunArgs(path string, load ConfigLoader) ([]string, string, error) {
	switch path {
	case "doctor":
		return []string{"doctor", "--offline"}, "gormes doctor --offline", nil
	case "auth status":
		if load == nil {
			load = func() (config.Config, error) { return config.Load(nil) }
		}
		cfg, err := load()
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
