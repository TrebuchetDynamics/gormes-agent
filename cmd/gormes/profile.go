package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

// profileCommandSeams collects every helper the gormes profile command
// surface consumes. Production wiring builds them from the existing CLI
// helpers; tests inject fakes. The seam shape mirrors the function-shaped
// seams that internal/cli/selector.go already exposes so the binding never
// re-derives validation, root resolution, or active-profile persistence.
type profileCommandSeams struct {
	ReadActiveProfileName cli.ReadActiveProfileNameFunc
	ValidateProfileName   cli.ValidateProfileNameFunc
	ResolveProfileRoot    cli.ResolveProfileRootFunc
	WriteActiveProfile    func(name string) error
	ListKnownProfiles     func() ([]string, error)
}

// Sentinel errors mirror the row's degraded_mode evidence codes. Callers
// branch with errors.Is; downstream rendering stays colocated with the
// command.
var (
	errProfileNameInvalid       = errors.New("profile_name_invalid")
	errProfileRootUnwritable    = errors.New("profile_root_unwritable")
	errActiveProfileCorrupt     = errors.New("active_profile_corrupt")
	errProfileSetPartialFailure = errors.New("profile_set_partial_failure")
)

const profileShowEllipsis = "..."

// newProfileCommand returns the production-wired `gormes profile` Cobra
// command. The seams are constructed from the same helpers internal/cli
// already exposes; profile create/delete/clone parity intentionally remains
// out of this slice.
func newProfileCommand() *cobra.Command {
	return newProfileCommandWithSeams(defaultProfileCommandSeams())
}

// newProfileCommandWithSeams returns a `gormes profile` command wired to the
// supplied seams. Tests inject fakes; production callers go through
// newProfileCommand.
func newProfileCommandWithSeams(seams profileCommandSeams) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "profile",
		Short:        "Inspect and switch the active Gormes profile",
		SilenceUsage: true,
	}
	cmd.AddCommand(newProfileShowCommand(seams))
	cmd.AddCommand(newProfileSetCommand(seams))
	cmd.AddCommand(newProfileListCommand(seams))
	return cmd
}

func newProfileShowCommand(seams profileCommandSeams) *cobra.Command {
	return &cobra.Command{
		Use:          "show",
		Short:        "Show the active Gormes profile and its redacted root path",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runProfileShowCommand(cmd, seams)
		},
	}
}

func newProfileSetCommand(seams profileCommandSeams) *cobra.Command {
	return &cobra.Command{
		Use:          "set <name>",
		Short:        "Switch the active Gormes profile by name",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileSetCommand(cmd, seams, args[0])
		},
	}
}

func newProfileListCommand(seams profileCommandSeams) *cobra.Command {
	return &cobra.Command{
		Use:          "list",
		Short:        "List known Gormes profiles and mark the active one",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runProfileListCommand(cmd, seams)
		},
	}
}

func runProfileShowCommand(cmd *cobra.Command, seams profileCommandSeams) error {
	selector := newProfileSelectorFromSeams(seams)
	profile, err := selector.Select(context.Background())
	if err != nil {
		if errors.Is(err, cli.ErrSelectorNoMatch) {
			fmt.Fprintln(cmd.OutOrStdout(), "active profile: <unset> (defaulting to 'default')")
			return nil
		}
		return fmt.Errorf("gormes profile show: %w: %w", errActiveProfileCorrupt, err)
	}
	writeProfileSummary(cmd, profile.Name, profile.RootPath)
	return nil
}

func runProfileSetCommand(cmd *cobra.Command, seams profileCommandSeams, rawName string) error {
	name := strings.TrimSpace(rawName)
	if seams.ValidateProfileName == nil || seams.WriteActiveProfile == nil || seams.ResolveProfileRoot == nil {
		return fmt.Errorf("gormes profile set: %w", cli.ErrSelectorHelperUnavailable)
	}
	if err := seams.ValidateProfileName(name); err != nil {
		return fmt.Errorf("gormes profile set %q: %w: %w", name, errProfileNameInvalid, err)
	}
	if err := seams.WriteActiveProfile(name); err != nil {
		return fmt.Errorf("gormes profile set %q: %w: %w", name, errProfileRootUnwritable, err)
	}
	root, err := seams.ResolveProfileRoot(name)
	if err != nil {
		return fmt.Errorf("gormes profile set %q: %w: %w", name, errProfileSetPartialFailure, err)
	}
	writeProfileSummary(cmd, name, root)
	return nil
}

// writeProfileSummary renders the canonical two-line profile summary used by
// both `show` and `set` so name/redacted-root rendering stays in one place.
func writeProfileSummary(cmd *cobra.Command, name, root string) {
	fmt.Fprintf(cmd.OutOrStdout(), "active profile: %s\n", name)
	fmt.Fprintf(cmd.OutOrStdout(), "root: %s\n", redactProfileRootPath(root))
}

func runProfileListCommand(cmd *cobra.Command, seams profileCommandSeams) error {
	if seams.ListKnownProfiles == nil {
		return fmt.Errorf("gormes profile list: %w", cli.ErrSelectorHelperUnavailable)
	}
	known, err := seams.ListKnownProfiles()
	if err != nil {
		return fmt.Errorf("gormes profile list: %w", err)
	}
	active := ""
	if seams.ReadActiveProfileName != nil {
		if name, err := seams.ReadActiveProfileName(); err == nil {
			active = strings.TrimSpace(name)
		}
	}
	if len(known) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "no profiles found")
		return nil
	}
	sorted := append([]string(nil), known...)
	sort.Strings(sorted)
	for _, name := range sorted {
		marker := " "
		if name == active {
			marker = "*"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", marker, name)
	}
	return nil
}

// redactProfileRootPath returns a bounded display form of a resolved profile
// root: the last path segment prefixed with an ellipsis. The full absolute
// path is never echoed because operator home directories can carry tokens or
// usernames that the row's degraded_mode forbids printing.
func redactProfileRootPath(root string) string {
	cleaned := strings.TrimSpace(root)
	if cleaned == "" {
		return profileShowEllipsis
	}
	last := filepath.Base(filepath.Clean(cleaned))
	if last == "" || last == "." || last == "/" {
		return profileShowEllipsis
	}
	return profileShowEllipsis + "/" + last
}

// newProfileSelectorFromSeams adapts the command's seams into the canonical
// internal/cli ProfileSelector. Keeping construction here means callers never
// import the selector package directly for production wiring.
func newProfileSelectorFromSeams(seams profileCommandSeams) cli.ProfileSelector {
	return cli.NewDefaultProfileSelector(cli.DefaultProfileSelectorOptions{
		ReadActiveProfileName: seams.ReadActiveProfileName,
		ValidateProfileName:   seams.ValidateProfileName,
		ResolveProfileRoot:    seams.ResolveProfileRoot,
	})
}

// defaultProfileCommandSeams wires the production helpers from internal/cli
// into profileCommandSeams. The active-profile file lives at
// GormesHome()/active_profile so the binding inherits whatever GORMES_HOME
// override is in effect for the process. Tests skip this and inject fakes via
// newProfileCommandWithSeams.
func defaultProfileCommandSeams() profileCommandSeams {
	activePath := filepath.Join(config.GormesHome(), "active_profile")
	xdgRoot := filepath.Dir(config.GormesHome())
	return profileCommandSeams{
		ReadActiveProfileName: func() (string, error) {
			return cli.ReadActiveProfile(activePath)
		},
		ValidateProfileName: cli.ValidateProfileName,
		ResolveProfileRoot: func(name string) (string, error) {
			return cli.ResolveProfileRoot(name, xdgRoot)
		},
		WriteActiveProfile: func(name string) error {
			return cli.WriteActiveProfile(activePath, name)
		},
		ListKnownProfiles: func() ([]string, error) {
			return defaultListKnownProfiles()
		},
	}
}

// defaultListKnownProfiles enumerates known profiles by reading the on-disk
// layout that ResolveProfileRoot produces. The default profile is always
// reported even if no profile dir exists yet so operators can always orient.
func defaultListKnownProfiles() ([]string, error) {
	known := []string{"default"}
	profilesDir := filepath.Join(config.GormesHome(), "profiles")
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		return known, nil
	}
	seen := map[string]struct{}{"default": {}}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		if err := cli.ValidateProfileName(name); err != nil {
			continue
		}
		seen[name] = struct{}{}
		known = append(known, name)
	}
	return known, nil
}
