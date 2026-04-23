package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/plugins"
)

var pluginsCmd = &cobra.Command{
	Use:   "plugins",
	Short: "Inspect manifest-backed third-party plugin extensions",
}

var pluginsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List discovered plugins and their activation state",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.Load(nil)
		if err != nil {
			return err
		}
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		entries, err := plugins.DiscoverCatalog(plugins.DiscoveryOptions{
			BundledRoots:    []string{filepath.Join(cwd, "plugins")},
			UserRoots:       []string{cfg.PluginsRoot()},
			ProjectRoots:    []string{filepath.Join(cwd, ".hermes", "plugins")},
			EnableProject:   projectPluginsEnabled(),
			DisabledGeneral: cfg.Plugins.Disabled,
			MemoryProvider:  cfg.Memory.Provider,
			ContextEngine:   cfg.Context.Engine,
		})
		if err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), "name\tkind\tsource\tstate\tdetails")
		for _, entry := range entries {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\t%s\n",
				entry.Name, entry.Kind, entry.Source, entry.State, entry.Details())
		}
		return nil
	},
}

func init() {
	pluginsCmd.AddCommand(pluginsListCmd)
}

func projectPluginsEnabled() bool {
	return envTruthy("GORMES_ENABLE_PROJECT_PLUGINS") || envTruthy("HERMES_ENABLE_PROJECT_PLUGINS")
}

func envTruthy(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
