package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/plugins"
)

func newPluginsCommand() *cobra.Command {
	return newPluginsCommandWithManager(defaultPluginLifecycleManager())
}

func defaultPluginLifecycleManager() *plugins.LifecycleManager {
	cwd, _ := os.Getwd()
	return plugins.NewLifecycleManager(plugins.LifecycleOptions{
		UserRoot:             filepath.Join(config.GormesHome(), "plugins"),
		ProjectRoot:          filepath.Join(cwd, ".gormes", "plugins"),
		Config:               config.ConfigPath(),
		CurrentGormesVersion: "1.0.0",
		EnableProjectPlugins: plugins.ProjectPluginsEnabledFromEnv(),
		Env:                  pluginDotEnv{path: config.EnvPath()},
		Prompt:               defaultPluginEnvPrompt,
	})
}

func newPluginsCommandWithManager(manager *plugins.LifecycleManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "plugins",
		Short:        "Manage Hermes-compatible plugins",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginsList(cmd, manager)
		},
	}
	cmd.AddCommand(newPluginsInstallCommand(manager))
	cmd.AddCommand(newPluginsListCommand(manager))
	cmd.AddCommand(newPluginsUpdateCommand(manager))
	cmd.AddCommand(newPluginsRemoveCommand(manager))
	cmd.AddCommand(newPluginsEnableCommand(manager))
	cmd.AddCommand(newPluginsDisableCommand(manager))
	return cmd
}

func newPluginsInstallCommand(manager *plugins.LifecycleManager) *cobra.Command {
	var force bool
	var enable bool
	cmd := &cobra.Command{
		Use:          "install <identifier>",
		Short:        "Install a plugin from a Git URL or owner/repo shorthand",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := manager.Install(args[0], plugins.InstallOptions{Force: force, Enable: enable})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "installed %s\n", result.Name)
			if enable {
				fmt.Fprintf(cmd.OutOrStdout(), "enabled %s\n", result.Name)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "not_enabled %s\n", result.Name)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "replace an existing plugin directory inside the plugin root")
	cmd.Flags().BoolVar(&enable, "enable", false, "enable the plugin after install")
	return cmd
}

func newPluginsListCommand(manager *plugins.LifecycleManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "list",
		Aliases:      []string{"ls"},
		Short:        "List installed plugins",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPluginsList(cmd, manager)
		},
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: `{build, plugins: [{name, version, status, source, path, description}]}`")
	return cmd
}

func newPluginsUpdateCommand(manager *plugins.LifecycleManager) *cobra.Command {
	return &cobra.Command{
		Use:          "update <name>",
		Short:        "Update a git-installed plugin",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := manager.Update(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "updated %s\n", args[0])
			return nil
		},
	}
}

func newPluginsRemoveCommand(manager *plugins.LifecycleManager) *cobra.Command {
	return &cobra.Command{
		Use:          "remove <name>",
		Aliases:      []string{"rm", "uninstall"},
		Short:        "Remove an installed plugin",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := manager.Remove(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed %s\n", args[0])
			return nil
		},
	}
}

func newPluginsEnableCommand(manager *plugins.LifecycleManager) *cobra.Command {
	return &cobra.Command{
		Use:          "enable <name>",
		Short:        "Enable an installed plugin",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := manager.Enable(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "enabled %s\n", args[0])
			return nil
		},
	}
}

func newPluginsDisableCommand(manager *plugins.LifecycleManager) *cobra.Command {
	return &cobra.Command{
		Use:          "disable <name>",
		Short:        "Disable an installed plugin",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := manager.Disable(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "disabled %s\n", args[0])
			return nil
		},
	}
}

func runPluginsList(cmd *cobra.Command, manager *plugins.LifecycleManager) error {
	entries, err := manager.List()
	if err != nil {
		return err
	}
	asJSON, _ := cmd.Flags().GetBool("json")
	if asJSON {
		report := pluginsListReportJSON{
			Build:   newBuildProvenance(),
			Plugins: make([]pluginsListEntryJSON, len(entries)),
		}
		for i, entry := range entries {
			report.Plugins[i] = pluginsListEntryJSON{
				Name:        entry.Name,
				Version:     entry.Version,
				Status:      string(entry.Status),
				Source:      string(entry.Source),
				Path:        entry.Path,
				Description: entry.Description,
			}
		}
		body, marshalErr := json.MarshalIndent(report, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(body))
		return nil
	}
	if len(entries) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No plugins installed.")
		fmt.Fprintln(cmd.OutOrStdout(), "Install with: gormes plugins install owner/repo")
		return nil
	}
	for _, entry := range entries {
		version := entry.Version
		if version == "" {
			version = "-"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", entry.Name, entry.Status, version, entry.Source)
	}
	return nil
}

// pluginsListReportJSON is the wire shape for `plugins list --json`.
// Fleet automation auditing plugin state across machines parses this
// to identify drift, missing plugins, or unexpected versions. Build
// provenance leads — same convention as the rest of the `--json` arc.
type pluginsListReportJSON struct {
	Build   buildProvenanceJSON    `json:"build"`
	Plugins []pluginsListEntryJSON `json:"plugins"`
}

type pluginsListEntryJSON struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	Source      string `json:"source"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

type pluginDotEnv struct {
	path string
}

func (e pluginDotEnv) Lookup(name string) (string, bool) {
	if value, ok := os.LookupEnv(name); ok && value != "" {
		return value, true
	}
	values, err := readPluginDotEnv(e.path)
	if err != nil {
		return "", false
	}
	value, ok := values[name]
	return value, ok && value != ""
}

func (e pluginDotEnv) Save(name, value string) error {
	values, err := readPluginDotEnv(e.path)
	if err != nil {
		return err
	}
	values[name] = value
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var body strings.Builder
	for _, key := range keys {
		body.WriteString(key)
		body.WriteByte('=')
		body.WriteString(values[key])
		body.WriteByte('\n')
	}
	if err := os.MkdirAll(filepath.Dir(e.path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(e.path, []byte(body.String()), 0o600)
}

func readPluginDotEnv(path string) (map[string]string, error) {
	values := make(map[string]string)
	if path == "" {
		return values, nil
	}
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return values, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return values, scanner.Err()
}

func defaultPluginEnvPrompt(name string) (string, bool, error) {
	info, err := os.Stdin.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return "", false, nil
	}
	fmt.Fprintf(os.Stderr, "%s: ", name)
	reader := bufio.NewReader(os.Stdin)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", false, err
	}
	return strings.TrimSpace(value), true, nil
}
