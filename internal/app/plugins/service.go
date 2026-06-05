package plugins

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

func DefaultLifecycleManager() *pluginspkg.LifecycleManager {
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

func Install(out io.Writer, manager *pluginspkg.LifecycleManager, identifier string, force, enable, asJSON bool, options Options) error {
	result, err := manager.Install(identifier, pluginspkg.InstallOptions{Force: force, Enable: enable})
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
		fmt.Fprintln(out, string(body))
		return nil
	}
	fmt.Fprintf(out, "installed %s\n", result.Name)
	if enable {
		fmt.Fprintf(out, "enabled %s\n", result.Name)
	} else {
		fmt.Fprintf(out, "not_enabled %s\n", result.Name)
	}
	return nil
}

type installReportJSON struct {
	Build   BuildProvenance `json:"build"`
	Action  string          `json:"action"`
	Name    string          `json:"name"`
	Path    string          `json:"path"`
	Enabled bool            `json:"enabled"`
}

func List(out io.Writer, manager *pluginspkg.LifecycleManager, asJSON bool, options Options) error {
	entries, err := manager.List()
	if err != nil {
		return err
	}
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
		fmt.Fprintln(out, string(body))
		return nil
	}
	if len(entries) == 0 {
		fmt.Fprintln(out, "No plugins installed.")
		fmt.Fprintln(out, "Install with: gormes plugins install owner/repo")
		return nil
	}
	for _, entry := range entries {
		version := entry.Version
		if version == "" {
			version = "-"
		}
		fmt.Fprintf(out, "%s\t%s\t%s\t%s\n", entry.Name, entry.Status, version, entry.Source)
	}
	return nil
}

func Update(out io.Writer, manager *pluginspkg.LifecycleManager, name string, asJSON bool, options Options) error {
	if _, err := manager.Update(name); err != nil {
		return err
	}
	return writeLifecycleResult(out, "updated", name, asJSON, options)
}

func Remove(out io.Writer, manager *pluginspkg.LifecycleManager, name string, asJSON bool, options Options) error {
	if err := manager.Remove(name); err != nil {
		return err
	}
	return writeLifecycleResult(out, "removed", name, asJSON, options)
}

func Enable(out io.Writer, manager *pluginspkg.LifecycleManager, name string, asJSON bool, options Options) error {
	if err := manager.Enable(name); err != nil {
		return err
	}
	return writeLifecycleResult(out, "enabled", name, asJSON, options)
}

func Disable(out io.Writer, manager *pluginspkg.LifecycleManager, name string, asJSON bool, options Options) error {
	if err := manager.Disable(name); err != nil {
		return err
	}
	return writeLifecycleResult(out, "disabled", name, asJSON, options)
}

type lifecycleReportJSON struct {
	Build  BuildProvenance `json:"build"`
	Action string          `json:"action"`
	Name   string          `json:"name"`
}

func writeLifecycleResult(out io.Writer, action, name string, asJSON bool, options Options) error {
	if asJSON {
		body, err := json.MarshalIndent(lifecycleReportJSON{
			Build:  options.buildProvenance(),
			Action: action,
			Name:   name,
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(body))
		return nil
	}
	fmt.Fprintf(out, "%s %s\n", action, name)
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
