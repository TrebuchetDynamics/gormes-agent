package plugins

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
	pluginspkg "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/plugins"
)

// BuildProvenance is the shared build metadata embedded in plugins JSON output.
type BuildProvenance struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

// Options supplies command-specific seams while keeping plugin behavior in this package.
type Options struct {
	BuildProvenance func() BuildProvenance
}

func (o Options) buildProvenance() BuildProvenance {
	if o.BuildProvenance != nil {
		return o.BuildProvenance()
	}
	return BuildProvenance{}
}

func NewCommand(options Options) *cobra.Command {
	return NewCommandWithManager(defaultLifecycleManager(), options)
}

func defaultLifecycleManager() *pluginspkg.LifecycleManager {
	cwd, _ := os.Getwd()
	return pluginspkg.NewLifecycleManager(pluginspkg.LifecycleOptions{
		UserRoot:             filepath.Join(config.GormesHome(), "plugins"),
		ProjectRoot:          filepath.Join(cwd, ".gormes", "plugins"),
		Config:               config.ConfigPath(),
		CurrentGormesVersion: "1.0.0",
		EnableProjectPlugins: pluginspkg.ProjectPluginsEnabledFromEnv(),
		Env:                  DotEnv{Path: config.EnvPath()},
		Prompt:               defaultEnvPrompt,
	})
}

func NewCommandWithManager(manager *pluginspkg.LifecycleManager, options Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "plugins",
		Short:        "Manage Hermes-compatible plugins",
		SilenceUsage: true,
		// NoArgs rejects positional args at the parent level. Without
		// it, a typo like `gormes plugins listt` silently fell through
		// to the parent's RunE and printed "No plugins installed." as
		// if the typo had succeeded; cobra was unable to surface its
		// typo suggestion because the parent had a RunE.
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, manager, options)
		},
	}
	cmd.AddCommand(newInstallCommand(manager, options))
	cmd.AddCommand(newListCommand(manager, options))
	cmd.AddCommand(newUpdateCommand(manager, options))
	cmd.AddCommand(newRemoveCommand(manager, options))
	cmd.AddCommand(newEnableCommand(manager, options))
	cmd.AddCommand(newDisableCommand(manager, options))
	return cmd
}

func newInstallCommand(manager *pluginspkg.LifecycleManager, options Options) *cobra.Command {
	var force bool
	var enable bool
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "install <identifier>",
		Short:        "Install a plugin from a Git URL or owner/repo shorthand",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := manager.Install(args[0], pluginspkg.InstallOptions{Force: force, Enable: enable})
			if err != nil {
				return err
			}
			if asJSON {
				body, marshalErr := json.MarshalIndent(installReportJSON{
					Build:   options.buildProvenance(),
					Action:  "installed",
					Name:    result.Name,
					Path:    result.Path,
					Enabled: enable,
				}, "", "  ")
				if marshalErr != nil {
					return marshalErr
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(body))
				return nil
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
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: {build, action: 'installed', name, path, enabled}")
	return cmd
}

type installReportJSON struct {
	Build   BuildProvenance `json:"build"`
	Action  string          `json:"action"`
	Name    string          `json:"name"`
	Path    string          `json:"path"`
	Enabled bool            `json:"enabled"`
}

func newListCommand(manager *pluginspkg.LifecycleManager, options Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "list",
		Aliases:      []string{"ls"},
		Short:        "List installed plugins",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, manager, options)
		},
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: `{build, plugins: [{name, version, status, source, path, description}]}`")
	return cmd
}

func newUpdateCommand(manager *pluginspkg.LifecycleManager, options Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "update <name>",
		Short:        "Update a git-installed plugin",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := manager.Update(args[0]); err != nil {
				return err
			}
			return writeLifecycleResult(cmd, "updated", args[0], options)
		},
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: {build, action: 'updated', name}")
	return cmd
}

func newRemoveCommand(manager *pluginspkg.LifecycleManager, options Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "remove <name>",
		Aliases:      []string{"rm", "uninstall"},
		Short:        "Remove an installed plugin",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := manager.Remove(args[0]); err != nil {
				return err
			}
			return writeLifecycleResult(cmd, "removed", args[0], options)
		},
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: {build, action: 'removed', name}")
	return cmd
}

func newEnableCommand(manager *pluginspkg.LifecycleManager, options Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "enable <name>",
		Short:        "Enable an installed plugin",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := manager.Enable(args[0]); err != nil {
				return err
			}
			return writeLifecycleResult(cmd, "enabled", args[0], options)
		},
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: {build, action: 'enabled', name}")
	return cmd
}

func newDisableCommand(manager *pluginspkg.LifecycleManager, options Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "disable <name>",
		Short:        "Disable an installed plugin",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := manager.Disable(args[0]); err != nil {
				return err
			}
			return writeLifecycleResult(cmd, "disabled", args[0], options)
		},
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: {build, action: 'disabled', name}")
	return cmd
}

type lifecycleReportJSON struct {
	Build  BuildProvenance `json:"build"`
	Action string          `json:"action"`
	Name   string          `json:"name"`
}

func writeLifecycleResult(cmd *cobra.Command, action, name string, options Options) error {
	asJSON, _ := cmd.Flags().GetBool("json")
	if asJSON {
		body, err := json.MarshalIndent(lifecycleReportJSON{
			Build:  options.buildProvenance(),
			Action: action,
			Name:   name,
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(body))
		return nil
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", action, name)
	return nil
}

func runList(cmd *cobra.Command, manager *pluginspkg.LifecycleManager, options Options) error {
	entries, err := manager.List()
	if err != nil {
		return err
	}
	asJSON, _ := cmd.Flags().GetBool("json")
	if asJSON {
		report := listReportJSON{
			Build:   options.buildProvenance(),
			Plugins: make([]listEntryJSON, len(entries)),
		}
		for i, entry := range entries {
			report.Plugins[i] = listEntryJSON{
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

type listReportJSON struct {
	Build   BuildProvenance `json:"build"`
	Plugins []listEntryJSON `json:"plugins"`
}

type listEntryJSON struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	Source      string `json:"source"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

type DotEnv struct {
	Path string
}

func (e DotEnv) Lookup(name string) (string, bool) {
	if value, ok := os.LookupEnv(name); ok && value != "" {
		return value, true
	}
	values, err := ReadDotEnv(e.Path)
	if err != nil {
		return "", false
	}
	value, ok := values[name]
	return value, ok && value != ""
}

func (e DotEnv) Save(name, value string) error {
	values, err := ReadDotEnv(e.Path)
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
	if err := os.MkdirAll(filepath.Dir(e.Path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(e.Path, []byte(body.String()), 0o600)
}

func ReadDotEnv(path string) (map[string]string, error) {
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

func defaultEnvPrompt(name string) (string, bool, error) {
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
