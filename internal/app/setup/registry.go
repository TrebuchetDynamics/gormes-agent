package setup

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress"
)

// SetupSection is feature-owned metadata for one `gormes setup <section>` entry.
type SetupSection struct {
	Name   string
	Label  string
	Module string
}

const (
	SetupModuleGateway   = progress.ModuleGateway
	SetupModuleNavivox   = progress.ModuleNavivox
	SetupModuleProviders = progress.ModuleProviders
	SetupModuleTools     = progress.ModuleTools
	SetupModuleTTS       = progress.ModuleTTS
	SetupModuleTUI       = progress.ModuleTUI
)

// SetupRegistry preserves ordered setup sections and exposes compatibility views.
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

// CanonicalSection resolves setup section aliases to the command's canonical section names.
func (r *SetupRegistry) CanonicalSection(section string) string {
	section = normalizeChoice(section)
	switch section {
	case "providers":
		return "provider"
	case "channel", "channels", "messaging", "messaging_platform", "messaging_platforms", "discord", "slack", "whatsapp":
		return "gateway"
	default:
		return section
	}
}

// KnownSection reports whether section is present in this setup registry.
func (r *SetupRegistry) KnownSection(section string) bool {
	if r == nil {
		return false
	}
	_, ok := r.labels[section]
	return ok
}

// SectionLabel returns the display label for section, or section itself when unknown.
func (r *SetupRegistry) SectionLabel(section string) string {
	if r == nil {
		return section
	}
	if label, ok := r.labels[section]; ok {
		return label
	}
	return section
}

// SectionPipeList renders setup section names in command-help pipe-list form.
func (r *SetupRegistry) SectionPipeList(start, end int) string {
	if r == nil {
		return ""
	}
	if start < 0 {
		start = 0
	}
	if end > len(r.sections) {
		end = len(r.sections)
	}
	if start >= end {
		return ""
	}
	names := make([]string, 0, end-start)
	for _, section := range r.sections[start:end] {
		names = append(names, section.Name)
	}
	return strings.Join(names, "|")
}

// SectionList renders every setup section name in command-help pipe-list form.
func (r *SetupRegistry) SectionList() string {
	if r == nil {
		return ""
	}
	return r.SectionPipeList(0, len(r.sections))
}

// SectionOwnership returns the source-parity ownership class for a setup section.
func SectionOwnership(section string) string {
	switch normalizeChoice(section) {
	case "model", "tts", "terminal", "gateway", "telegram", "tools", "agent":
		return "hermes_owned"
	case "provider", "workspace", "bindings", "navivox", "router":
		return "gormes_owned_extension"
	default:
		return "unknown"
	}
}

type setupContractRegistry interface {
	ValidateSetupSections([]string) error
	SetupSectionOwner(string) (string, bool)
}

// ValidateContracts proves every setup section is owned by the same feature
// module declared in the CLI contract registry.
func (r *SetupRegistry) ValidateContracts(contracts setupContractRegistry) error {
	if r == nil || contracts == nil {
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

func normalizeToken(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
