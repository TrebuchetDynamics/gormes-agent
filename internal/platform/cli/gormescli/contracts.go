// Package gormescli owns the importable command-contract registry for the
// gormes binary. The runtime Cobra tree still lives in cmd/gormes during the
// first migration slice; this package records module ownership and validates
// that the live command tree, setup sections, and slash registry do not drift.
package gormescli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
)

// ModuleContract is the command/control-plane ownership declaration for one
// approved Gormes feature module.
type ModuleContract struct {
	Module        string
	Commands      []CommandSpec
	SetupSections []SetupSectionSpec
	SlashCommands []SlashCommandSpec
	JSONReports   []ReportSpec
}

// CommandSpec declares ownership for a Cobra command path. When
// IncludeDescendants is true, the module also owns every visible descendant
// path below Path.
type CommandSpec struct {
	Path               string
	IncludeDescendants bool
	ReadOnly           bool
	JSON               bool
	TestCommands       []string
}

// SetupSectionSpec declares ownership for `gormes setup <section>` sections.
type SetupSectionSpec struct {
	Name string
}

// SlashCommandSpec declares ownership for an internal slash command name.
type SlashCommandSpec struct {
	Name string
}

// ReportSpec is intentionally lightweight in the first slice. Ownership is
// enforced now; report-shape metadata can harden after the registry lands.
type ReportSpec struct {
	Name string
	Path string
}

// CommandManifestEntry is the exact command path -> module read model derived
// from the live Cobra tree and this registry.
type CommandManifestEntry struct {
	Path   string
	Module string
}

// Registry indexes module contracts and answers ownership queries.
type Registry struct {
	modules       []ModuleContract
	commandSpecs  []ownedCommandSpec
	setupOwners   map[string]string
	slashOwners   map[string]string
	modulePresent map[string]struct{}
}

type ownedCommandSpec struct {
	module string
	spec   CommandSpec
}

// NewRegistry validates and indexes module contracts.
func NewRegistry(modules []ModuleContract) (*Registry, error) {
	r := &Registry{
		setupOwners:   map[string]string{},
		slashOwners:   map[string]string{},
		modulePresent: map[string]struct{}{},
	}
	for _, module := range modules {
		module.Module = normalizeToken(module.Module)
		if !progress.ValidModule(module.Module) {
			return nil, fmt.Errorf("invalid module %q", module.Module)
		}
		if _, exists := r.modulePresent[module.Module]; exists {
			return nil, fmt.Errorf("duplicate module contract %q", module.Module)
		}
		r.modulePresent[module.Module] = struct{}{}
		r.modules = append(r.modules, module)

		for _, spec := range module.Commands {
			spec.Path = normalizePath(spec.Path)
			if spec.Path == "" {
				return nil, fmt.Errorf("module %q has empty command path", module.Module)
			}
			owned := ownedCommandSpec{module: module.Module, spec: spec}
			for _, prior := range r.commandSpecs {
				if commandSpecsOverlap(prior.spec, spec) {
					return nil, fmt.Errorf("command ownership overlap: %q (%s) and %q (%s)",
						prior.spec.Path, prior.module, spec.Path, module.Module)
				}
			}
			r.commandSpecs = append(r.commandSpecs, owned)
		}

		for _, section := range module.SetupSections {
			name := normalizeToken(section.Name)
			if name == "" {
				return nil, fmt.Errorf("module %q has empty setup section", module.Module)
			}
			if prior, exists := r.setupOwners[name]; exists {
				return nil, fmt.Errorf("setup section %q owned by both %s and %s", name, prior, module.Module)
			}
			r.setupOwners[name] = module.Module
		}

		for _, slash := range module.SlashCommands {
			name := normalizeSlash(slash.Name)
			if name == "" {
				return nil, fmt.Errorf("module %q has empty slash command", module.Module)
			}
			if prior, exists := r.slashOwners[name]; exists {
				return nil, fmt.Errorf("slash command %q owned by both %s and %s", name, prior, module.Module)
			}
			r.slashOwners[name] = module.Module
		}
	}
	sort.Slice(r.commandSpecs, func(i, j int) bool {
		return r.commandSpecs[i].spec.Path < r.commandSpecs[j].spec.Path
	})
	return r, nil
}

// Modules returns the registered module IDs in deterministic order.
func (r *Registry) Modules() []string {
	if r == nil {
		return nil
	}
	out := make([]string, 0, len(r.modulePresent))
	for module := range r.modulePresent {
		out = append(out, module)
	}
	sort.Strings(out)
	return out
}

// CommandOwner returns the unique module owner for a command path.
func (r *Registry) CommandOwner(path string) (string, bool) {
	if r == nil {
		return "", false
	}
	path = normalizePath(path)
	var owner string
	for _, spec := range r.commandSpecs {
		if commandSpecOwns(spec.spec, path) {
			if owner != "" {
				return "", false
			}
			owner = spec.module
		}
	}
	return owner, owner != ""
}

// SetupSectionOwner returns the unique module owner for a setup section.
func (r *Registry) SetupSectionOwner(name string) (string, bool) {
	if r == nil {
		return "", false
	}
	owner, ok := r.setupOwners[normalizeToken(name)]
	return owner, ok
}

// SlashCommandOwner returns the unique module owner for a slash command name.
func (r *Registry) SlashCommandOwner(name string) (string, bool) {
	if r == nil {
		return "", false
	}
	owner, ok := r.slashOwners[normalizeSlash(name)]
	return owner, ok
}

// CommandManifest maps live Cobra paths to module owners and fails closed when
// a path has no owner.
func (r *Registry) CommandManifest(paths []string) ([]CommandManifestEntry, error) {
	out := make([]CommandManifestEntry, 0, len(paths))
	for _, raw := range paths {
		path := normalizePath(raw)
		owner, ok := r.CommandOwner(path)
		if !ok {
			return nil, fmt.Errorf("no module owner for command path %q", path)
		}
		out = append(out, CommandManifestEntry{Path: path, Module: owner})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// ValidateSetupSections fails when a setup section is not feature-owned.
func (r *Registry) ValidateSetupSections(sections []string) error {
	for _, section := range sections {
		name := normalizeToken(section)
		if _, ok := r.SetupSectionOwner(name); !ok {
			return fmt.Errorf("no module owner for setup section %q", name)
		}
	}
	return nil
}

// ValidateSlashCommands fails when a slash command is not feature-owned.
func (r *Registry) ValidateSlashCommands(names []string) error {
	for _, name := range names {
		slash := normalizeSlash(name)
		if _, ok := r.SlashCommandOwner(slash); !ok {
			return fmt.Errorf("no module owner for slash command %q", slash)
		}
	}
	return nil
}

func commandSpecsOverlap(a, b CommandSpec) bool {
	a.Path = normalizePath(a.Path)
	b.Path = normalizePath(b.Path)
	if a.Path == b.Path {
		return true
	}
	return (a.IncludeDescendants && strings.HasPrefix(b.Path, a.Path+" ")) ||
		(b.IncludeDescendants && strings.HasPrefix(a.Path, b.Path+" "))
}

func commandSpecOwns(spec CommandSpec, path string) bool {
	spec.Path = normalizePath(spec.Path)
	path = normalizePath(path)
	if spec.Path == path {
		return true
	}
	return spec.IncludeDescendants && strings.HasPrefix(path, spec.Path+" ")
}

func normalizePath(path string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(path)), " ")
}

func normalizeToken(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeSlash(value string) string {
	value = normalizeToken(value)
	return strings.TrimPrefix(value, "/")
}
