// Package plugins provides the Phase 5.I plugin SDK baseline: typed manifest
// parsing plus deterministic on-disk discovery for future runtime wiring.
package plugins

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const manifestFilename = "plugin.yaml"

// Kind identifies the plugin family a manifest belongs to.
type Kind string

const (
	KindGeneral        Kind = "general"
	KindMemoryProvider Kind = "memory_provider"
	KindContextEngine  Kind = "context_engine"
)

// EnvRequirement describes one environment variable a plugin needs at runtime.
// It accepts either a bare string or a rich YAML object in plugin.yaml.
type EnvRequirement struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	URL         string `yaml:"url,omitempty" json:"url,omitempty"`
	Secret      bool   `yaml:"secret,omitempty" json:"secret,omitempty"`
}

func (r *EnvRequirement) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		var name string
		if err := node.Decode(&name); err != nil {
			return err
		}
		r.Name = strings.TrimSpace(name)
		return nil
	case yaml.MappingNode:
		type raw EnvRequirement
		var decoded raw
		if err := node.Decode(&decoded); err != nil {
			return err
		}
		*r = EnvRequirement(decoded)
		r.Name = strings.TrimSpace(r.Name)
		r.Description = strings.TrimSpace(r.Description)
		r.URL = strings.TrimSpace(r.URL)
		return nil
	default:
		return fmt.Errorf("requires_env entries must be strings or mappings")
	}
}

// Manifest is the typed plugin.yaml contract future runtime loaders consume.
type Manifest struct {
	Name             string           `yaml:"name" json:"name"`
	Version          string           `yaml:"version" json:"version"`
	Description      string           `yaml:"description,omitempty" json:"description,omitempty"`
	Author           string           `yaml:"author,omitempty" json:"author,omitempty"`
	ProvidesTools    []string         `yaml:"provides_tools,omitempty" json:"provides_tools,omitempty"`
	ProvidesHooks    []string         `yaml:"provides_hooks,omitempty" json:"provides_hooks,omitempty"`
	ProvidesSkills   []string         `yaml:"provides_skills,omitempty" json:"provides_skills,omitempty"`
	ProvidesCommands []string         `yaml:"provides_commands,omitempty" json:"provides_commands,omitempty"`
	RequiresEnv      []EnvRequirement `yaml:"requires_env,omitempty" json:"requires_env,omitempty"`

	Kind         Kind   `yaml:"-" json:"kind"`
	RootDir      string `yaml:"-" json:"root_dir"`
	ManifestPath string `yaml:"-" json:"manifest_path"`
}

// LoadManifest reads one plugin manifest from a directory or a plugin.yaml path.
func LoadManifest(path string) (Manifest, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return Manifest{}, errors.New("plugins: manifest path is required")
	}

	info, err := os.Stat(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("plugins: stat %q: %w", path, err)
	}

	rootDir := path
	manifestPath := path
	if info.IsDir() {
		rootDir = path
		manifestPath = filepath.Join(path, manifestFilename)
	} else {
		rootDir = filepath.Dir(path)
	}

	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return Manifest{}, fmt.Errorf("plugins: read manifest %q: %w", manifestPath, err)
	}

	var manifest Manifest
	if err := yaml.Unmarshal(raw, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("plugins: parse manifest %q: %w", manifestPath, err)
	}
	manifest.Name = strings.TrimSpace(manifest.Name)
	manifest.Version = strings.TrimSpace(manifest.Version)
	manifest.Description = strings.TrimSpace(manifest.Description)
	manifest.Author = strings.TrimSpace(manifest.Author)
	manifest.ProvidesTools = normalizeStrings(manifest.ProvidesTools)
	manifest.ProvidesHooks = normalizeStrings(manifest.ProvidesHooks)
	manifest.ProvidesSkills = normalizeStrings(manifest.ProvidesSkills)
	manifest.ProvidesCommands = normalizeStrings(manifest.ProvidesCommands)
	manifest.Kind = KindGeneral
	manifest.RootDir = rootDir
	manifest.ManifestPath = manifestPath

	if err := manifest.Validate(); err != nil {
		return Manifest{}, fmt.Errorf("plugins: invalid manifest %q: %w", manifestPath, err)
	}
	return manifest, nil
}

// Discover reads every immediate child directory under root that contains a
// plugin.yaml manifest. Results are sorted deterministically by manifest name.
func Discover(root string, kind Kind) ([]Manifest, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("plugins: read root %q: %w", root, err)
	}

	kind = normalizeKind(kind)
	out := make([]Manifest, 0, len(entries))
	seen := make(map[string]string, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dir := filepath.Join(root, entry.Name())
		manifestPath := filepath.Join(dir, manifestFilename)
		if _, err := os.Stat(manifestPath); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("plugins: stat manifest %q: %w", manifestPath, err)
		}

		manifest, err := LoadManifest(manifestPath)
		if err != nil {
			return nil, err
		}
		manifest.Kind = kind
		if prior, ok := seen[manifest.Name]; ok {
			return nil, fmt.Errorf("plugins: duplicate manifest name %q in %q and %q", manifest.Name, prior, manifestPath)
		}
		seen[manifest.Name] = manifestPath
		out = append(out, manifest)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		if out[i].Version != out[j].Version {
			return out[i].Version < out[j].Version
		}
		return out[i].ManifestPath < out[j].ManifestPath
	})
	return out, nil
}

// Validate enforces the minimal manifest contract the Go runtime depends on.
func (m Manifest) Validate() error {
	var errs []error
	if m.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}
	if m.Version == "" {
		errs = append(errs, errors.New("version is required"))
	}
	for _, field := range []struct {
		name   string
		values []string
	}{
		{name: "provides_tools", values: m.ProvidesTools},
		{name: "provides_hooks", values: m.ProvidesHooks},
		{name: "provides_skills", values: m.ProvidesSkills},
		{name: "provides_commands", values: m.ProvidesCommands},
	} {
		for i, value := range field.values {
			if strings.TrimSpace(value) == "" {
				errs = append(errs, fmt.Errorf("%s[%d] must be non-empty", field.name, i))
			}
		}
	}
	for i, req := range m.RequiresEnv {
		if strings.TrimSpace(req.Name) == "" {
			errs = append(errs, fmt.Errorf("requires_env[%d].name is required", i))
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func normalizeKind(kind Kind) Kind {
	switch strings.TrimSpace(string(kind)) {
	case "", string(KindGeneral):
		return KindGeneral
	case string(KindMemoryProvider):
		return KindMemoryProvider
	case string(KindContextEngine):
		return KindContextEngine
	default:
		return kind
	}
}

func normalizeStrings(values []string) []string {
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = strings.TrimSpace(value)
	}
	return out
}
