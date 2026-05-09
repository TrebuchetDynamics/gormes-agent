package plugins

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type PluginLifecycleStatus string

const (
	PluginLifecycleStatusEnabled    PluginLifecycleStatus = "enabled"
	PluginLifecycleStatusDisabled   PluginLifecycleStatus = "disabled"
	PluginLifecycleStatusNotEnabled PluginLifecycleStatus = "not_enabled"
)

type LifecycleOptions struct {
	UserRoot             string
	ProjectRoot          string
	Config               string
	CurrentGormesVersion string
	EnableProjectPlugins bool
	Runner               LifecycleRunner
	Env                  LifecycleEnv
	Prompt               LifecyclePrompt
}

type LifecycleRunner interface {
	Clone(url, dst string) error
	Pull(dir string) (string, error)
}

type LifecycleEnv interface {
	Lookup(name string) (string, bool)
	Save(name, value string) error
}

type LifecyclePrompt func(name string) (value string, ok bool, err error)

type LifecycleManager struct {
	userRoot             string
	projectRoot          string
	configPath           string
	currentGormesVersion string
	enableProjectPlugins bool
	runner               LifecycleRunner
	env                  LifecycleEnv
	prompt               LifecyclePrompt
}

type InstallOptions struct {
	Force  bool
	Enable bool
}

type InstalledPlugin struct {
	Name string
	Path string
}

type PluginLifecycleEntry struct {
	Name        string
	Version     string
	Description string
	Source      Source
	Status      PluginLifecycleStatus
	Path        string
}

type PluginConfigSets struct {
	Enabled  []string `toml:"enabled"`
	Disabled []string `toml:"disabled"`
}

func NewLifecycleManager(opts LifecycleOptions) *LifecycleManager {
	runner := opts.Runner
	if runner == nil {
		runner = gitLifecycleRunner{}
	}
	version := opts.CurrentGormesVersion
	if version == "" {
		version = defaultGormesVersion
	}
	return &LifecycleManager{
		userRoot:             opts.UserRoot,
		projectRoot:          opts.ProjectRoot,
		configPath:           opts.Config,
		currentGormesVersion: version,
		enableProjectPlugins: opts.EnableProjectPlugins,
		runner:               runner,
		env:                  opts.Env,
		prompt:               opts.Prompt,
	}
}

func (m *LifecycleManager) PluginPath(name string) (string, error) {
	root := filepath.Clean(m.userRoot)
	if root == "." || root == "" {
		return "", errors.New("plugin_cli_root_missing: user plugin root is empty")
	}
	clean := strings.TrimSpace(name)
	if !pluginNameRE.MatchString(clean) || strings.Contains(clean, "..") || strings.ContainsAny(clean, `/\`) {
		return "", fmt.Errorf("plugin_cli_invalid_name: invalid plugin name %q", name)
	}
	target := filepath.Clean(filepath.Join(root, clean))
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." {
		return "", fmt.Errorf("plugin_cli_invalid_name: plugin name %q escapes plugin root", name)
	}
	return target, nil
}

func (m *LifecycleManager) Install(identifier string, opts InstallOptions) (InstalledPlugin, error) {
	url, err := ResolvePluginIdentifier(identifier)
	if err != nil {
		return InstalledPlugin{}, err
	}
	if err := os.MkdirAll(m.userRoot, 0o755); err != nil {
		return InstalledPlugin{}, fmt.Errorf("plugin_cli_root_unwritable: %w", err)
	}
	tmp, err := os.MkdirTemp("", "gormes-plugin-*")
	if err != nil {
		return InstalledPlugin{}, err
	}
	defer os.RemoveAll(tmp)
	cloneDir := filepath.Join(tmp, "plugin")
	if err := m.runner.Clone(url, cloneDir); err != nil {
		return InstalledPlugin{}, fmt.Errorf("plugin_cli_clone_failed: %w", err)
	}
	status := LoadDir(cloneDir, LoadOptions{
		Source:               SourceUser,
		CurrentGormesVersion: m.currentGormesVersion,
		EnvLookup:            m.envHas,
	})
	name := strings.TrimSpace(status.Manifest.Name)
	if name == "" {
		name = RepoNameFromPluginURL(url)
	}
	target, err := m.PluginPath(name)
	if err != nil {
		return InstalledPlugin{}, err
	}
	if status.State == StateInvalid || status.State == StateMalformed {
		return InstalledPlugin{}, fmt.Errorf("plugin_cli_invalid_manifest: plugin %q failed manifest validation: %s", name, lifecycleEvidenceSummary(status.Evidence))
	}
	if _, statErr := os.Stat(target); statErr == nil {
		if !opts.Force {
			return InstalledPlugin{}, fmt.Errorf("plugin_cli_already_exists: plugin %q already exists", name)
		}
		if err := os.RemoveAll(target); err != nil {
			return InstalledPlugin{}, fmt.Errorf("plugin_cli_remove_failed: %w", err)
		}
	} else if !os.IsNotExist(statErr) {
		return InstalledPlugin{}, statErr
	}
	if err := moveDir(cloneDir, target); err != nil {
		return InstalledPlugin{}, fmt.Errorf("plugin_cli_install_failed: %w", err)
	}
	if err := copyExampleFiles(target); err != nil {
		return InstalledPlugin{}, err
	}
	if err := m.promptRequiredEnv(status.Manifest.RequiresEnv); err != nil {
		return InstalledPlugin{}, err
	}
	if opts.Enable {
		if err := m.Enable(name); err != nil {
			return InstalledPlugin{}, err
		}
	}
	return InstalledPlugin{Name: name, Path: target}, nil
}

func (m *LifecycleManager) List() ([]PluginLifecycleEntry, error) {
	sets, err := m.ConfigSets()
	if err != nil {
		return nil, err
	}
	inventory := Discover(DiscoveryRoots{
		User:    []string{m.userRoot},
		Project: m.projectRoot,
	}, DiscoverOptions{
		CurrentGormesVersion: m.currentGormesVersion,
		EnvLookup:            m.envHas,
		EnableProjectPlugins: m.enableProjectPlugins,
	})
	enabled := stringSet(sets.Enabled)
	disabled := stringSet(sets.Disabled)
	entries := make([]PluginLifecycleEntry, 0, len(inventory.Plugins))
	for _, status := range inventory.Plugins {
		state := PluginLifecycleStatusNotEnabled
		if disabled[status.Name] {
			state = PluginLifecycleStatusDisabled
		} else if enabled[status.Name] {
			state = PluginLifecycleStatusEnabled
		}
		entries = append(entries, PluginLifecycleEntry{
			Name:        status.Name,
			Version:     status.Version,
			Description: status.Description,
			Source:      status.Source,
			Status:      state,
			Path:        filepath.Join(m.userRoot, status.Name),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Name != entries[j].Name {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Source < entries[j].Source
	})
	return entries, nil
}

func (m *LifecycleManager) Update(name string) (bool, error) {
	target, err := m.requireInstalled(name)
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(filepath.Join(target, ".git")); err != nil {
		return false, fmt.Errorf("plugin_cli_update_not_git: plugin %q was not installed from git", name)
	}
	if _, err := m.runner.Pull(target); err != nil {
		return false, fmt.Errorf("plugin_cli_update_failed: %w", err)
	}
	if err := copyExampleFiles(target); err != nil {
		return false, err
	}
	return true, nil
}

func (m *LifecycleManager) Remove(name string) error {
	target, err := m.requireInstalled(name)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("plugin_cli_remove_failed: %w", err)
	}
	sets, err := m.ConfigSets()
	if err != nil {
		return err
	}
	sets.Enabled = removeString(sets.Enabled, name)
	sets.Disabled = removeString(sets.Disabled, name)
	return m.writeConfigSets(sets)
}

func (m *LifecycleManager) Enable(name string) error {
	if _, err := m.findPlugin(name); err != nil {
		return err
	}
	sets, err := m.ConfigSets()
	if err != nil {
		return err
	}
	sets.Enabled = addString(sets.Enabled, name)
	sets.Disabled = removeString(sets.Disabled, name)
	return m.writeConfigSets(sets)
}

func (m *LifecycleManager) Disable(name string) error {
	if _, err := m.findPlugin(name); err != nil {
		return err
	}
	sets, err := m.ConfigSets()
	if err != nil {
		return err
	}
	sets.Enabled = removeString(sets.Enabled, name)
	sets.Disabled = addString(sets.Disabled, name)
	return m.writeConfigSets(sets)
}

func (m *LifecycleManager) ConfigSets() (PluginConfigSets, error) {
	if m.configPath == "" {
		return PluginConfigSets{}, nil
	}
	body, err := os.ReadFile(m.configPath)
	if os.IsNotExist(err) {
		return PluginConfigSets{}, nil
	}
	if err != nil {
		return PluginConfigSets{}, err
	}
	var decoded struct {
		Plugins PluginConfigSets `toml:"plugins"`
	}
	if err := toml.Unmarshal(body, &decoded); err != nil {
		return PluginConfigSets{}, fmt.Errorf("plugin_cli_config_invalid: %w", err)
	}
	decoded.Plugins.Enabled = normalizeStrings(decoded.Plugins.Enabled)
	decoded.Plugins.Disabled = normalizeStrings(decoded.Plugins.Disabled)
	return decoded.Plugins, nil
}

func (m *LifecycleManager) requireInstalled(name string) (string, error) {
	target, err := m.PluginPath(name)
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(target); err != nil || !info.IsDir() {
		if err == nil {
			err = fmt.Errorf("not a directory")
		}
		return "", fmt.Errorf("plugin_cli_not_found: plugin %q not found: %w", name, err)
	}
	return target, nil
}

func (m *LifecycleManager) findPlugin(name string) (string, error) {
	target, err := m.PluginPath(name)
	if err == nil {
		if info, statErr := os.Stat(target); statErr == nil && info.IsDir() {
			return target, nil
		}
	}
	for _, entry := range []string{m.userRoot, m.projectRoot} {
		if entry == "" {
			continue
		}
		statuses := discoverRoot(entry, LoadOptions{Source: SourceUser, CurrentGormesVersion: m.currentGormesVersion})
		for _, status := range statuses {
			if status.Name == name {
				return filepath.Join(entry, name), nil
			}
		}
	}
	return "", fmt.Errorf("plugin_cli_not_found: plugin %q is not installed", name)
}

func (m *LifecycleManager) envHas(name string) bool {
	if m.env == nil {
		return os.Getenv(name) != ""
	}
	_, ok := m.env.Lookup(name)
	return ok
}

func (m *LifecycleManager) promptRequiredEnv(names []string) error {
	if m.env == nil || m.prompt == nil {
		return nil
	}
	for _, name := range normalizeStrings(names) {
		if _, ok := m.env.Lookup(name); ok {
			continue
		}
		value, ok, err := m.prompt(name)
		if err != nil {
			return err
		}
		if !ok || strings.TrimSpace(value) == "" {
			continue
		}
		if err := m.env.Save(name, strings.TrimSpace(value)); err != nil {
			return err
		}
	}
	return nil
}

func (m *LifecycleManager) writeConfigSets(sets PluginConfigSets) error {
	if m.configPath == "" {
		return nil
	}
	sets.Enabled = normalizeStrings(sets.Enabled)
	sets.Disabled = normalizeStrings(sets.Disabled)
	body, err := os.ReadFile(m.configPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	next := upsertPluginsConfig(body, sets)
	if err := os.MkdirAll(filepath.Dir(m.configPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(m.configPath, next, 0o600)
}

func ResolvePluginIdentifier(identifier string) (string, error) {
	value := strings.TrimSpace(identifier)
	if strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "git@") || strings.HasPrefix(value, "ssh://") || strings.HasPrefix(value, "file://") {
		return value, nil
	}
	parts := strings.Split(strings.Trim(value, "/"), "/")
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return "https://github.com/" + parts[0] + "/" + parts[1] + ".git", nil
	}
	return "", fmt.Errorf("plugin_cli_invalid_identifier: use a Git URL or owner/repo shorthand")
}

func lifecycleEvidenceSummary(evidence []Evidence) string {
	if len(evidence) == 0 {
		return "no evidence"
	}
	parts := make([]string, 0, len(evidence))
	for _, item := range evidence {
		if item.Field != "" {
			parts = append(parts, item.Code+":"+item.Field)
		} else {
			parts = append(parts, item.Code)
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

func RepoNameFromPluginURL(url string) string {
	value := strings.TrimSuffix(strings.TrimRight(url, "/"), ".git")
	value = strings.TrimPrefix(value, "file://")
	if idx := strings.LastIndex(value, "/"); idx >= 0 {
		value = value[idx+1:]
	}
	if idx := strings.LastIndex(value, ":"); idx >= 0 {
		value = value[idx+1:]
	}
	return strings.TrimSpace(value)
}

type gitLifecycleRunner struct{}

func (gitLifecycleRunner) Clone(url, dst string) error {
	cmd := exec.Command("git", "clone", "--depth", "1", url, dst)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (gitLifecycleRunner) Pull(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "pull", "--ff-only")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func copyExampleFiles(dir string) error {
	matches, err := filepath.Glob(filepath.Join(dir, "*.example"))
	if err != nil {
		return err
	}
	for _, example := range matches {
		target := strings.TrimSuffix(example, ".example")
		if _, err := os.Stat(target); err == nil {
			continue
		}
		body, err := os.ReadFile(example)
		if err != nil {
			return err
		}
		if err := os.WriteFile(target, body, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func moveDir(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := copyDir(src, dst); err != nil {
		return err
	}
	return os.RemoveAll(src)
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, body, 0o644)
	})
}

func upsertPluginsConfig(body []byte, sets PluginConfigSets) []byte {
	var kept []string
	inPlugins := false
	for _, line := range strings.Split(string(body), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			inPlugins = trimmed == "[plugins]"
			if inPlugins {
				continue
			}
		}
		if !inPlugins && trimmed != "" {
			kept = append(kept, line)
		}
	}
	var out bytes.Buffer
	for _, line := range kept {
		out.WriteString(line)
		out.WriteByte('\n')
	}
	if out.Len() > 0 {
		out.WriteByte('\n')
	}
	out.WriteString("[plugins]\n")
	out.WriteString("enabled = ")
	out.WriteString(tomlStringList(sets.Enabled))
	out.WriteByte('\n')
	out.WriteString("disabled = ")
	out.WriteString(tomlStringList(sets.Disabled))
	out.WriteByte('\n')
	return out.Bytes()
}

func tomlStringList(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range normalizeStrings(values) {
		quoted = append(quoted, strconv.Quote(value))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func normalizeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func stringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

func addString(values []string, value string) []string {
	return normalizeStrings(append(values, value))
}

func removeString(values []string, value string) []string {
	out := values[:0]
	for _, item := range values {
		if item != value {
			out = append(out, item)
		}
	}
	return normalizeStrings(out)
}
