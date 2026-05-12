package main

import (
	"context"
	"encoding/json"
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
	ReadActiveProfileName    cli.ReadActiveProfileNameFunc
	ValidateProfileName      cli.ValidateProfileNameFunc
	ResolveProfileRoot       cli.ResolveProfileRootFunc
	WriteActiveProfile       func(name string) error
	CreateProfile            func(name string, cloneAll bool) (cli.ProfileCreateResult, error)
	ListKnownProfiles        func() ([]string, error)
	ReadDistributionManifest func(root string) (cli.ProfileDistributionManifest, bool, error)
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
		Args:         cobra.NoArgs,
	}
	cmd.AddCommand(newProfileShowCommand(seams))
	cmd.AddCommand(newProfileSetCommand(seams))
	cmd.AddCommand(newProfileCreateCommand(seams))
	cmd.AddCommand(newProfileListCommand(seams))
	cmd.AddCommand(newProfileInfoCommand(seams))
	return cmd
}

func newProfileShowCommand(seams profileCommandSeams) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "show",
		Short:        "Show the active Gormes profile and its redacted root path",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runProfileShowCommand(cmd, seams, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit a `{build, active, root}` JSON document with the same redacted root path the human surface prints")
	return cmd
}

func newProfileSetCommand(seams profileCommandSeams) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "set <name>",
		Short:        "Switch the active Gormes profile by name",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileSetCommand(cmd, seams, args[0])
		},
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: `{build, action, active, root}` with the same redacted root path as `profile show`")
	return cmd
}

// profileSetReportJSON is the wire shape for `profile set --json`.
// Fleet automation switching profiles parses this to confirm the
// active marker landed. Root is redacted (only the trailing segment)
// — same secrets contract as `profile show`.
type profileSetReportJSON struct {
	Build  buildProvenanceJSON `json:"build"`
	Action string              `json:"action"`
	Active string              `json:"active"`
	Root   string              `json:"root"`
}

func newProfileCreateCommand(seams profileCommandSeams) *cobra.Command {
	var cloneAll bool
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "create <name>",
		Short:        "Create a named Gormes profile",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileCreateCommand(cmd, seams, args[0], cloneAll, asJSON)
		},
	}
	cmd.Flags().BoolVar(&cloneAll, "clone-all", false, "copy the default profile minus infrastructure and runtime files")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: `{build, action, name, root, clone_all}` with a redacted root")
	return cmd
}

type profileCreateReportJSON struct {
	Build    buildProvenanceJSON `json:"build"`
	Action   string              `json:"action"`
	Name     string              `json:"name"`
	Root     string              `json:"root"`
	CloneAll bool                `json:"clone_all"`
}

func newProfileInfoCommand(seams profileCommandSeams) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "info <name>",
		Short:        "Show a profile distribution manifest",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileInfoCommand(cmd, seams, args[0], asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit a `{build, name, root, distribution}` JSON document with the profile root redacted")
	return cmd
}

func newProfileListCommand(seams profileCommandSeams) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List known Gormes profiles and mark the active one",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runProfileListCommand(cmd, seams, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit a `{build, active, profiles: [...]}` JSON document (suitable for fleet inventory automation)")
	return cmd
}

func runProfileShowCommand(cmd *cobra.Command, seams profileCommandSeams, asJSON bool) error {
	selector := newProfileSelectorFromSeams(seams)
	profile, err := selector.Select(context.Background())
	if err != nil {
		if errors.Is(err, cli.ErrSelectorNoMatch) {
			if asJSON {
				return emitProfileShowJSON(cmd, "", "", nil)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "active profile: <unset> (defaulting to 'default')")
			return nil
		}
		return fmt.Errorf("gormes profile show: %w: %w", errActiveProfileCorrupt, err)
	}
	manifest, hasManifest, err := readProfileDistributionForRoot(seams, profile.RootPath)
	if err != nil {
		return fmt.Errorf("gormes profile show: %w", err)
	}
	if asJSON {
		return emitProfileShowJSON(cmd, profile.Name, redactProfileRootPath(profile.RootPath), manifestPointer(manifest, hasManifest))
	}
	writeProfileSummary(cmd, profile.Name, profile.RootPath)
	if hasManifest {
		writeProfileDistributionSummary(cmd, manifest)
	}
	return nil
}

// profileShowReportJSON is the wire shape for `profile show --json`.
// Build provenance leads, then the active marker and the redacted root
// — same convention as the rest of the --json arc. `active` is empty
// when no profile is set (the human surface emits "<unset>").
type profileShowReportJSON struct {
	Build        buildProvenanceJSON             `json:"build"`
	Active       string                          `json:"active"`
	Root         string                          `json:"root"`
	Distribution *profileDistributionSummaryJSON `json:"distribution,omitempty"`
}

func emitProfileShowJSON(cmd *cobra.Command, active, root string, manifest *cli.ProfileDistributionManifest) error {
	body, err := json.MarshalIndent(profileShowReportJSON{
		Build:        newBuildProvenance(),
		Active:       active,
		Root:         root,
		Distribution: profileDistributionSummaryJSONFromManifest(manifest),
	}, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(body))
	return err
}

func runProfileSetCommand(cmd *cobra.Command, seams profileCommandSeams, rawName string) error {
	name := strings.TrimSpace(rawName)
	if seams.ValidateProfileName == nil || seams.WriteActiveProfile == nil || seams.ResolveProfileRoot == nil {
		return fmt.Errorf("gormes profile set: %w", cli.ErrSelectorHelperUnavailable)
	}
	if err := seams.ValidateProfileName(name); err != nil {
		return fmt.Errorf("gormes profile set %q: %w: %w", name, errProfileNameInvalid, err)
	}
	// Refuse names not in the known-profiles list so operators don't
	// silently end up pointing at a non-existent profile root. Without
	// this guard a typo like `gormes profile set deafult` would write
	// the marker, and every subsequent command would either fail with
	// a cryptic "no such file or directory" or silently fall back to
	// default (confusing). Only enforced when ListKnownProfiles is
	// wired so test seams that stub the selector keep working.
	if seams.ListKnownProfiles != nil {
		known, err := seams.ListKnownProfiles()
		if err == nil {
			recognized := false
			for _, k := range known {
				if k == name {
					recognized = true
					break
				}
			}
			if !recognized {
				return fmt.Errorf("gormes profile set %q: unknown profile (known: %s)",
					name, strings.Join(known, ", "))
			}
		}
	}
	if err := seams.WriteActiveProfile(name); err != nil {
		return fmt.Errorf("gormes profile set %q: %w: %w", name, errProfileRootUnwritable, err)
	}
	root, err := seams.ResolveProfileRoot(name)
	if err != nil {
		return fmt.Errorf("gormes profile set %q: %w: %w", name, errProfileSetPartialFailure, err)
	}
	asJSON, _ := cmd.Flags().GetBool("json")
	if asJSON {
		body, marshalErr := json.MarshalIndent(profileSetReportJSON{
			Build:  newBuildProvenance(),
			Action: "set",
			Active: name,
			Root:   redactProfileRootPath(root),
		}, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), string(body))
		return err
	}
	writeProfileSummary(cmd, name, root)
	return nil
}

func runProfileCreateCommand(cmd *cobra.Command, seams profileCommandSeams, rawName string, cloneAll bool, asJSON bool) error {
	name := strings.TrimSpace(rawName)
	if seams.CreateProfile == nil {
		return fmt.Errorf("gormes profile create: %w", cli.ErrSelectorHelperUnavailable)
	}
	result, err := seams.CreateProfile(name, cloneAll)
	if err != nil {
		return fmt.Errorf("gormes profile create %q: %w", name, err)
	}
	redactedRoot := redactProfileRootPath(result.Root)
	if asJSON {
		body, marshalErr := json.MarshalIndent(profileCreateReportJSON{
			Build:    newBuildProvenance(),
			Action:   "created",
			Name:     result.Name,
			Root:     redactedRoot,
			CloneAll: result.CloneAll,
		}, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), string(body))
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "created profile: %s\n", result.Name)
	fmt.Fprintf(cmd.OutOrStdout(), "root: %s\n", redactedRoot)
	if result.CloneAll {
		fmt.Fprintln(cmd.OutOrStdout(), "clone_all: true")
	}
	return nil
}

// writeProfileSummary renders the canonical two-line profile summary used by
// both `show` and `set` so name/redacted-root rendering stays in one place.
func writeProfileSummary(cmd *cobra.Command, name, root string) {
	fmt.Fprintf(cmd.OutOrStdout(), "active profile: %s\n", name)
	fmt.Fprintf(cmd.OutOrStdout(), "root: %s\n", redactProfileRootPath(root))
}

func runProfileListCommand(cmd *cobra.Command, seams profileCommandSeams, asJSON bool) error {
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
	sorted := append([]string(nil), known...)
	sort.Strings(sorted)
	if asJSON {
		profiles := make([]profileListEntryJSON, len(sorted))
		for i, name := range sorted {
			manifest, hasManifest, err := readProfileDistributionForName(seams, name)
			if err != nil {
				return fmt.Errorf("gormes profile list %q: %w", name, err)
			}
			profiles[i] = profileListEntryJSON{
				Name:         name,
				Active:       name == active,
				Distribution: profileDistributionSummaryJSONFromManifest(manifestPointer(manifest, hasManifest)),
			}
		}
		body, err := json.MarshalIndent(profileListReportJSON{
			Build:    newBuildProvenance(),
			Active:   active,
			Profiles: profiles,
		}, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), string(body))
		return err
	}
	if len(known) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "no profiles found")
		return nil
	}
	for _, name := range sorted {
		marker := " "
		if name == active {
			marker = "*"
		}
		manifest, hasManifest, err := readProfileDistributionForName(seams, name)
		if err != nil {
			return fmt.Errorf("gormes profile list %q: %w", name, err)
		}
		if hasManifest {
			fmt.Fprintf(cmd.OutOrStdout(), "%s %-24s %s\n", marker, name, manifest.Summary())
			continue
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", marker, name)
	}
	return nil
}

func runProfileInfoCommand(cmd *cobra.Command, seams profileCommandSeams, rawName string, asJSON bool) error {
	name := strings.TrimSpace(rawName)
	if seams.ValidateProfileName == nil || seams.ResolveProfileRoot == nil || seams.ReadDistributionManifest == nil {
		return fmt.Errorf("gormes profile info: %w", cli.ErrSelectorHelperUnavailable)
	}
	if err := seams.ValidateProfileName(name); err != nil {
		return fmt.Errorf("gormes profile info %q: %w: %w", name, errProfileNameInvalid, err)
	}
	root, err := seams.ResolveProfileRoot(name)
	if err != nil {
		return fmt.Errorf("gormes profile info %q: %w", name, err)
	}
	manifest, ok, err := seams.ReadDistributionManifest(root)
	if err != nil {
		return fmt.Errorf("gormes profile info %q: %w", name, err)
	}
	if asJSON {
		return emitProfileInfoJSON(cmd, name, redactProfileRootPath(root), manifestPointer(manifest, ok))
	}
	if !ok {
		fmt.Fprintf(cmd.OutOrStdout(), "Profile '%s' is not a distribution (no %s).\n", name, cli.ProfileDistributionManifestFile)
		return nil
	}
	writeProfileDistributionInfo(cmd, manifest)
	return nil
}

// profileListReportJSON is the wire shape for `profile list --json`.
// Build provenance leads, then the active marker and the profile array
// — same convention as update / doctor / status / restore / auth /
// gateway-status / secrets.
type profileListReportJSON struct {
	Build    buildProvenanceJSON    `json:"build"`
	Active   string                 `json:"active"`
	Profiles []profileListEntryJSON `json:"profiles"`
}

type profileListEntryJSON struct {
	Name         string                          `json:"name"`
	Active       bool                            `json:"active"`
	Distribution *profileDistributionSummaryJSON `json:"distribution,omitempty"`
}

type profileInfoReportJSON struct {
	Build        buildProvenanceJSON              `json:"build"`
	Name         string                           `json:"name"`
	Root         string                           `json:"root"`
	Distribution *cli.ProfileDistributionManifest `json:"distribution,omitempty"`
}

func emitProfileInfoJSON(cmd *cobra.Command, name, root string, manifest *cli.ProfileDistributionManifest) error {
	body, err := json.MarshalIndent(profileInfoReportJSON{
		Build:        newBuildProvenance(),
		Name:         name,
		Root:         root,
		Distribution: manifest,
	}, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(body))
	return err
}

type profileDistributionSummaryJSON struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Source  string `json:"source,omitempty"`
}

func readProfileDistributionForName(seams profileCommandSeams, name string) (cli.ProfileDistributionManifest, bool, error) {
	if seams.ResolveProfileRoot == nil || seams.ReadDistributionManifest == nil {
		return cli.ProfileDistributionManifest{}, false, nil
	}
	root, err := seams.ResolveProfileRoot(name)
	if err != nil {
		return cli.ProfileDistributionManifest{}, false, err
	}
	return readProfileDistributionForRoot(seams, root)
}

func readProfileDistributionForRoot(seams profileCommandSeams, root string) (cli.ProfileDistributionManifest, bool, error) {
	if seams.ReadDistributionManifest == nil {
		return cli.ProfileDistributionManifest{}, false, nil
	}
	manifest, ok, err := seams.ReadDistributionManifest(root)
	if err != nil {
		return cli.ProfileDistributionManifest{}, false, err
	}
	return manifest, ok, nil
}

func manifestPointer(manifest cli.ProfileDistributionManifest, ok bool) *cli.ProfileDistributionManifest {
	if !ok {
		return nil
	}
	return &manifest
}

func profileDistributionSummaryJSONFromManifest(manifest *cli.ProfileDistributionManifest) *profileDistributionSummaryJSON {
	if manifest == nil || strings.TrimSpace(manifest.Name) == "" {
		return nil
	}
	return &profileDistributionSummaryJSON{
		Name:    manifest.Name,
		Version: manifest.Version,
		Source:  manifest.Source,
	}
}

func writeProfileDistributionSummary(cmd *cobra.Command, manifest cli.ProfileDistributionManifest) {
	if summary := manifest.Summary(); summary != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "distribution: %s\n", summary)
	}
}

func writeProfileDistributionInfo(cmd *cobra.Command, manifest cli.ProfileDistributionManifest) {
	fmt.Fprintf(cmd.OutOrStdout(), "\nDistribution: %s\n", manifest.Name)
	fmt.Fprintf(cmd.OutOrStdout(), "Version:      %s\n", manifest.Version)
	if manifest.Description != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Description:  %s\n", manifest.Description)
	}
	if manifest.Author != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Author:       %s\n", manifest.Author)
	}
	if manifest.License != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "License:      %s\n", manifest.License)
	}
	if manifest.HermesRequires != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Requires:     Hermes %s\n", manifest.HermesRequires)
	}
	if manifest.Source != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Source:       %s\n", manifest.Source)
	}
	if manifest.InstalledAt != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Installed:    %s\n", manifest.InstalledAt)
	}
	if len(manifest.EnvRequires) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "\nEnvironment variables:")
		for _, req := range manifest.EnvRequires {
			tag := "required"
			if !req.Required {
				tag = "optional"
			}
			line := fmt.Sprintf("  %s (%s)", req.Name, tag)
			if req.Description != "" {
				line += " - " + req.Description
			}
			fmt.Fprintln(cmd.OutOrStdout(), line)
			if req.Default != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "      default: %s\n", *req.Default)
			}
		}
	}
	fmt.Fprintln(cmd.OutOrStdout())
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
	return profileCommandSeams{
		ReadActiveProfileName: func() (string, error) {
			return cli.ReadActiveProfile(activePath)
		},
		ValidateProfileName: cli.ValidateProfileName,
		ResolveProfileRoot: func(name string) (string, error) {
			if name == "default" {
				return config.GormesHome(), nil
			}
			if err := cli.ValidateProfileName(name); err != nil {
				return "", err
			}
			return filepath.Join(config.GormesHome(), "profiles", name), nil
		},
		WriteActiveProfile: func(name string) error {
			return cli.WriteActiveProfile(activePath, name)
		},
		CreateProfile: func(name string, cloneAll bool) (cli.ProfileCreateResult, error) {
			if name == "default" {
				return cli.ProfileCreateResult{}, cli.ErrProfileCreateDefaultReserved
			}
			return cli.CreateProfile(cli.ProfileCreateOptions{
				Name:       name,
				TargetRoot: filepath.Join(config.GormesHome(), "profiles", name),
				SourceRoot: config.GormesHome(),
				CloneAll:   cloneAll,
			})
		},
		ListKnownProfiles: func() ([]string, error) {
			return defaultListKnownProfiles()
		},
		ReadDistributionManifest: cli.ReadProfileDistributionManifest,
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
