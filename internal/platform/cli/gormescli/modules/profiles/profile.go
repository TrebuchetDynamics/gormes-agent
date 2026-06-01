package profiles

import (
	"github.com/spf13/cobra"

	profilecommand "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/profiles/command"
)

// Seams collects every helper the gormes profile command surface consumes.
// The command implementation lives in the focused command package; this alias
// preserves the profiles module as the public CLI boundary for existing callers.
type Seams = profilecommand.Seams

// Options carries binary-owned process metadata into the importable profile
// command module.
type Options = profilecommand.Options

// NewCommand returns the production-wired `gormes profile` Cobra command.
func NewCommand(opts ...Options) *cobra.Command {
	return profilecommand.NewCommand(opts...)
}

// NewCommandWithSeams returns a `gormes profile` command wired to the supplied
// seams. Tests inject fakes; production callers go through NewCommand.
func NewCommandWithSeams(seams Seams, opts ...Options) *cobra.Command {
	return profilecommand.NewCommandWithSeams(seams, opts...)
}

// DefaultSeams wires the production helpers from internal/cli into Seams.
func DefaultSeams() Seams {
	return profilecommand.DefaultSeams()
}

// DefaultListKnownProfiles enumerates known profiles from the root config
// registry and on-disk profile layout.
func DefaultListKnownProfiles() ([]string, error) {
	return profilecommand.DefaultListKnownProfiles()
}
