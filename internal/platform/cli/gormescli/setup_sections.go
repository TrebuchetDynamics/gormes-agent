package gormescli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// setupRegistry is the canonical setup section registry, initialized by
// InitSetupRegistry which must be called once from the root main package
// before any setup commands run.
var setupRegistry *SetupRegistry

// InitSetupRegistry initializes the global setup registry with the given
// section definitions. Must be called once from cmd/gormes/main.go before
// any setup command registration. Not safe for concurrent use.
func InitSetupRegistry(sections []SetupSection) {
	setupRegistry = MustSetupRegistry(sections)
}

// SetupSectionNames returns all registered setup section names.
func SetupSectionNames() []string {
	if setupRegistry == nil {
		return nil
	}
	return setupRegistry.Names()
}

// SetupSectionLabels returns the section → label map.
func SetupSectionLabels() map[string]string {
	if setupRegistry == nil {
		return nil
	}
	return setupRegistry.Labels()
}

// SetupCanonicalSection returns the canonical section name for the given input.
func SetupCanonicalSection(section string) string {
	if setupRegistry == nil {
		return section
	}
	return setupRegistry.CanonicalSection(section)
}

// SetupKnownSection reports whether the given section name is registered.
func SetupKnownSection(section string) bool {
	if setupRegistry == nil {
		return false
	}
	return setupRegistry.KnownSection(section)
}

// SetupSectionLabel returns the human-readable label for a section.
func SetupSectionLabel(section string) string {
	if setupRegistry == nil {
		return section
	}
	return setupRegistry.SectionLabel(section)
}

// SetupSectionPipeList returns a pipe-separated list of section names in a range.
func SetupSectionPipeList(start, end int) string {
	if setupRegistry == nil {
		return ""
	}
	return setupRegistry.SectionPipeList(start, end)
}

// SetupSectionUnsupported writes an unsupported-section error message.
func SetupSectionUnsupported(cmd *cobra.Command, section string) error {
	fmt.Fprintf(cmd.OutOrStdout(), "Unsupported setup section: %s\n", section)
	fmt.Fprintf(cmd.OutOrStdout(), "Run `gormes setup` for available sections.\n")
	return nil
}

// SetupSectionList returns a formatted section list for error messages.
func SetupSectionList() string {
	sections := SetupSectionNames()
	if len(sections) == 0 {
		return "no sections registered"
	}
	return strings.Join(sections, ", ")
}

// PrintSetupSections prints the available setup sections and quick-start commands.
func PrintSetupSections(cmd *cobra.Command) {
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Available setup sections:")
	if setupRegistry != nil {
		for _, section := range setupRegistry.Sections() {
			fmt.Fprintf(out, "  - %-10s %s\n", section.Name, section.Label)
		}
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Quick starts:")
	fmt.Fprintln(out, "  Interactive menu: gormes setup")
	fmt.Fprintln(out, "  Terminal/TUI quick setup: gormes setup --quick --target tui")
	fmt.Fprintln(out, "  Provider setup: gormes setup provider")
	fmt.Fprintln(out, "  Telegram setup: gormes setup telegram")
	fmt.Fprintln(out, "  Router setup:   gormes setup router")
}