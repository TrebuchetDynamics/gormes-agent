package gormescli

import (
	"fmt"

	"github.com/spf13/cobra"

	pluginsapp "github.com/TrebuchetDynamics/gormes-agent/internal/app/plugins"
	pluginspkg "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/plugins"
)

type PluginsBuildProvenance struct{ Version, GitCommit string }

type PluginsOptions struct{ BuildProvenance func() PluginsBuildProvenance }

func NewPluginsCommand(options PluginsOptions) *cobra.Command {
	return NewPluginsCommandWithManager(pluginsapp.DefaultLifecycleManager(), options)
}

func NewPluginsCommandWithManager(manager any, options PluginsOptions) *cobra.Command {
	m, ok := manager.(*pluginspkg.LifecycleManager)
	if !ok {
		panic(fmt.Sprintf("plugins command manager must be *plugins.LifecycleManager, got %T", manager))
	}
	appOptions := pluginsapp.Options{BuildProvenance: func() pluginsapp.BuildProvenance {
		if options.BuildProvenance == nil {
			return pluginsapp.BuildProvenance{}
		}
		build := options.BuildProvenance()
		return pluginsapp.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
	}}
	return newPluginsCommandWithManagerOptions(m, appOptions)
}

func newPluginsCommandWithManagerOptions(manager *pluginspkg.LifecycleManager, options pluginsapp.Options) *cobra.Command {
	cmd := &cobra.Command{Use: "plugins", Short: "Manage Hermes-compatible plugins", SilenceUsage: true, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		return pluginsapp.List(cmd.OutOrStdout(), manager, false, options)
	}}
	cmd.AddCommand(newPluginsInstallCommand(manager, options), newPluginsListCommand(manager, options), newPluginsUpdateCommand(manager, options), newPluginsRemoveCommand(manager, options), newPluginsEnableCommand(manager, options), newPluginsDisableCommand(manager, options))
	return cmd
}

func newPluginsInstallCommand(manager *pluginspkg.LifecycleManager, options pluginsapp.Options) *cobra.Command {
	var force, enable, asJSON bool
	cmd := &cobra.Command{Use: "install <identifier>", Short: "Install a plugin from a Git URL or owner/repo shorthand", Args: cobra.ExactArgs(1), SilenceUsage: true, RunE: func(cmd *cobra.Command, args []string) error {
		return pluginsapp.Install(cmd.OutOrStdout(), manager, args[0], force, enable, asJSON, options)
	}}
	cmd.Flags().BoolVar(&force, "force", false, "replace an existing plugin directory inside the plugin root")
	cmd.Flags().BoolVar(&enable, "enable", false, "enable the plugin after install")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: {build, action: 'installed', name, path, enabled}")
	return cmd
}

func newPluginsListCommand(manager *pluginspkg.LifecycleManager, options pluginsapp.Options) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{Use: "list", Aliases: []string{"ls"}, Short: "List installed plugins", SilenceUsage: true, RunE: func(cmd *cobra.Command, args []string) error {
		return pluginsapp.List(cmd.OutOrStdout(), manager, asJSON, options)
	}}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: `{build, plugins: [{name, version, status, source, path, description}]}`")
	return cmd
}

func newPluginsUpdateCommand(manager *pluginspkg.LifecycleManager, options pluginsapp.Options) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{Use: "update <name>", Short: "Update a git-installed plugin", Args: cobra.ExactArgs(1), SilenceUsage: true, RunE: func(cmd *cobra.Command, args []string) error {
		return pluginsapp.Update(cmd.OutOrStdout(), manager, args[0], asJSON, options)
	}}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: {build, action: 'updated', name}")
	return cmd
}

func newPluginsRemoveCommand(manager *pluginspkg.LifecycleManager, options pluginsapp.Options) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{Use: "remove <name>", Aliases: []string{"rm", "uninstall"}, Short: "Remove an installed plugin", Args: cobra.ExactArgs(1), SilenceUsage: true, RunE: func(cmd *cobra.Command, args []string) error {
		return pluginsapp.Remove(cmd.OutOrStdout(), manager, args[0], asJSON, options)
	}}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: {build, action: 'removed', name}")
	return cmd
}

func newPluginsEnableCommand(manager *pluginspkg.LifecycleManager, options pluginsapp.Options) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{Use: "enable <name>", Short: "Enable an installed plugin", Args: cobra.ExactArgs(1), SilenceUsage: true, RunE: func(cmd *cobra.Command, args []string) error {
		return pluginsapp.Enable(cmd.OutOrStdout(), manager, args[0], asJSON, options)
	}}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: {build, action: 'enabled', name}")
	return cmd
}

func newPluginsDisableCommand(manager *pluginspkg.LifecycleManager, options pluginsapp.Options) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{Use: "disable <name>", Short: "Disable an installed plugin", Args: cobra.ExactArgs(1), SilenceUsage: true, RunE: func(cmd *cobra.Command, args []string) error {
		return pluginsapp.Disable(cmd.OutOrStdout(), manager, args[0], asJSON, options)
	}}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: {build, action: 'disabled', name}")
	return cmd
}
