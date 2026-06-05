package contractruntime

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
)

// SetupSection is the feature-owned metadata for one `gormes setup <section>`
// entry. Runtime dispatch can stay in cmd/gormes while section identity,
// labels, order, and module ownership move into the importable CLI layer.
type SetupSection struct {
	Name   string
	Label  string
	Module string
}

// Setup module constants keep command-surface metadata in the CLI layer so
// cmd/gormes does not need to import progress schema details for static setup
// section ownership.
const (
	SetupModuleGateway   = progress.ModuleGateway
	SetupModuleNavivox   = progress.ModuleNavivox
	SetupModuleProviders = progress.ModuleProviders
	SetupModuleTools     = progress.ModuleTools
	SetupModuleTTS       = progress.ModuleTTS
	SetupModuleTUI       = progress.ModuleTUI
)

// SetupRegistry preserves ordered setup sections and exposes compatibility
// views for the existing cmd/gormes setup code.
type SetupRegistry struct {
	sections []SetupSection
	labels   map[string]string
}

// NewSetupRegistry validates setup section metadata and preserves order.
func NewSetupRegistry(sections []SetupSection) (*SetupRegistry, error) {
	r := &SetupRegistry{
		sections: make([]SetupSection, 0, len(sections)),
		labels:   map[string]string{},
	}
	seen := map[string]struct{}{}
	for _, section := range sections {
		section.Name = normalizeToken(section.Name)
		section.Label = strings.TrimSpace(section.Label)
		section.Module = normalizeToken(section.Module)
		if section.Name == "" {
			return nil, fmt.Errorf("setup registry: empty section name")
		}
		if section.Label == "" {
			return nil, fmt.Errorf("setup registry: section %q has empty label", section.Name)
		}
		if !progress.ValidModule(section.Module) {
			return nil, fmt.Errorf("setup registry: section %q has invalid module %q", section.Name, section.Module)
		}
		if _, ok := seen[section.Name]; ok {
			return nil, fmt.Errorf("setup registry: duplicate section %q", section.Name)
		}
		seen[section.Name] = struct{}{}
		r.sections = append(r.sections, section)
		r.labels[section.Name] = section.Label
	}
	return r, nil
}

// MustSetupRegistry panics on invalid static setup metadata.
func MustSetupRegistry(sections []SetupSection) *SetupRegistry {
	registry, err := NewSetupRegistry(sections)
	if err != nil {
		panic(err)
	}
	return registry
}

// Sections returns a copy of the ordered setup section metadata.
func (r *SetupRegistry) Sections() []SetupSection {
	if r == nil {
		return nil
	}
	return append([]SetupSection(nil), r.sections...)
}

// Names returns setup section names in presentation order.
func (r *SetupRegistry) Names() []string {
	if r == nil {
		return nil
	}
	out := make([]string, 0, len(r.sections))
	for _, section := range r.sections {
		out = append(out, section.Name)
	}
	return out
}

// Labels returns the setup section label map used by boxed setup chrome.
func (r *SetupRegistry) Labels() map[string]string {
	if r == nil {
		return nil
	}
	out := make(map[string]string, len(r.labels))
	for name, label := range r.labels {
		out[name] = label
	}
	return out
}

// ValidateContracts proves every setup section is owned by the same feature
// module declared in the CLI contract registry.
func (r *SetupRegistry) ValidateContracts(contracts *Registry) error {
	if r == nil {
		return nil
	}
	if err := contracts.ValidateSetupSections(r.Names()); err != nil {
		return err
	}
	for _, section := range r.sections {
		owner, ok := contracts.SetupSectionOwner(section.Name)
		if !ok {
			return fmt.Errorf("setup registry: section %q has no contract owner", section.Name)
		}
		if owner != section.Module {
			return fmt.Errorf("setup registry: section %q module %q does not match contract owner %q", section.Name, section.Module, owner)
		}
	}
	return nil
}
