package profiles

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

	"github.com/TrebuchetDynamics/gormes-agent/internal/app/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

// Seams collects every helper the gormes profile command
// surface consumes. Production wiring builds them from the existing CLI
// helpers; tests inject fakes. The seam shape mirrors the function-shaped
// seams that internal/cli/selector.go already exposes so the binding never
// re-derives validation, root resolution, or active-profile persistence.
type Seams struct {
	ReadActiveProfileName    cli.ReadActiveProfileNameFunc
	ValidateProfileName      cli.ValidateProfileNameFunc
	ResolveProfileRoot       cli.ResolveProfileRootFunc
	WriteActiveProfile       func(name string) error
	CreateProfile            func(name string, cloneAll bool) (cli.ProfileCreateResult, error)
	ListKnownProfiles        func() ([]string, error)
	ReadDistributionManifest func(root string) (cli.ProfileDistributionManifest, bool, error)
}

// Options carries binary-owned process metadata into the importable profile
// command module. cmd/gormes supplies release/version values; tests can inject
// stable values without importing the main package.
type Options struct {
	BuildProvenance func() gormescli.BuildProvenance
}

func normalizeOptions(opts []Options) Options {
	var out Options
	if len(opts) > 0 {
		out = opts[0]
	}
	if out.BuildProvenance == nil {
		out.BuildProvenance = func() gormescli.BuildProvenance {
			return gormescli.BuildProvenance{Version: "unknown", GitCommit: "unknown"}
		}
	}
	return out
}

func buildProvenance(options Options) gormescli.BuildProvenance {
	options = normalizeOptions([]Options{options})
	return options.BuildProvenance()
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

// NewCommand returns the production-wired `gormes profile` Cobra
// command. The seams are constructed from the same helpers internal/cli
// already exposes; profile create/delete/clone parity intentionally remains
// out of this slice.
func NewCommand(opts ...Options) *cobra.Command {
	return NewCommandWithSeams(DefaultSeams(), opts...)
}

// NewCommandWithSeams returns a `gormes profile` command wired to the
// supplied seams. Tests inject fakes; production callers go through
// NewCommand.
func NewCommandWithSeams(seams Seams, opts ...Options) *cobra.Command {
	options := normalizeOptions(opts)
	cmd := &cobra.Command{
		Use:          "profile",
		Short:        "Inspect and switch the active Gormes profile",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
	}
	cmd.AddCommand(newProfileListCommand(seams, options))
	cmd.AddCommand(newProfileUseCommand(seams, options))
	cmd.AddCommand(newProfileCreateCommand(seams, options))
	cmd.AddCommand(newProfileUnavailableCommand(profileUnavailableSpec{
		Name:        "delete",
		Use:         "delete <name>",
		Short:       "Delete a named Gormes profile",
		Args:        cobra.ExactArgs(1),
		Destructive: true,
		FlagSet:     profileUnavailableDeleteFlags,
	}, options))
	cmd.AddCommand(newProfileShowCommand(seams, options))
	cmd.AddCommand(newProfileUnavailableCommand(profileUnavailableSpec{
		Name:    "alias",
		Use:     "alias <name>",
		Short:   "Manage profile wrapper aliases",
		Args:    cobra.ExactArgs(1),
		FlagSet: profileUnavailableAliasFlags,
	}, options))
	cmd.AddCommand(newProfileUnavailableCommand(profileUnavailableSpec{
		Name:    "rename",
		Use:     "rename <old-name> <new-name>",
		Short:   "Rename a Gormes profile",
		Args:    cobra.ExactArgs(2),
		FlagSet: profileUnavailableJSONFlag,
	}, options))
	cmd.AddCommand(newProfileUnavailableCommand(profileUnavailableSpec{
		Name:    "export",
		Use:     "export <name>",
		Short:   "Export a profile archive",
		Args:    cobra.ExactArgs(1),
		FlagSet: profileUnavailableExportFlags,
	}, options))
	cmd.AddCommand(newProfileUnavailableCommand(profileUnavailableSpec{
		Name:    "import",
		Use:     "import <archive>",
		Short:   "Import a profile archive",
		Args:    cobra.ExactArgs(1),
		FlagSet: profileUnavailableImportFlags,
	}, options))
	cmd.AddCommand(newProfileUnavailableCommand(profileUnavailableSpec{
		Name:    "install",
		Use:     "install <source>",
		Short:   "Install a profile distribution",
		Args:    cobra.ExactArgs(1),
		FlagSet: profileUnavailableInstallFlags,
	}, options))
	cmd.AddCommand(newProfileUnavailableCommand(profileUnavailableSpec{
		Name:    "update",
		Use:     "update <name>",
		Short:   "Update a profile distribution",
		Args:    cobra.ExactArgs(1),
		FlagSet: profileUnavailableUpdateFlags,
	}, options))
	cmd.AddCommand(newProfileInfoCommand(seams, options))
	return cmd
}

func newProfileShowCommand(seams Seams, options Options) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "show",
		Short:        "Show the active Gormes profile and its redacted root path",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runProfileShowCommand(cmd, seams, asJSON, options)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit a `{build, active, root}` JSON document with the same redacted root path the human surface prints")
	return cmd
}

func newProfileUseCommand(seams Seams, options Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "use <name>",
		Aliases:      []string{"set"},
		Short:        "Switch the active Gormes profile by name",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			action := "use"
			if cmd.CalledAs() == "set" {
				action = "set"
			}
			return runProfileSetCommand(cmd, seams, args[0], action, options)
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
	Build  gormescli.BuildProvenance `json:"build"`
	Action string                    `json:"action"`
	Active string                    `json:"active"`
	Root   string                    `json:"root"`
}

func newProfileCreateCommand(seams Seams, options Options) *cobra.Command {
	var cloneAll bool
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "create <name>",
		Short:        "Create a named Gormes profile",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileCreateCommand(cmd, seams, args[0], cloneAll, asJSON, options)
		},
	}
	cmd.Flags().BoolVar(&cloneAll, "clone-all", false, "copy the default profile minus infrastructure and runtime files")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: `{build, action, name, root, clone_all}` with a redacted root")
	return cmd
}

type profileCreateReportJSON struct {
	Build    gormescli.BuildProvenance `json:"build"`
	Action   string                    `json:"action"`
	Name     string                    `json:"name"`
	Root     string                    `json:"root"`
	CloneAll bool                      `json:"clone_all"`
}

type profileUnavailableSpec struct {
	Name        string
	Use         string
	Short       string
	Args        cobra.PositionalArgs
	Destructive bool
	FlagSet     func(*cobra.Command)
}

type profileUnavailableReportJSON struct {
	Build       gormescli.BuildProvenance `json:"build"`
	Action      string                    `json:"action"`
	Command     string                    `json:"command"`
	Status      string                    `json:"status"`
	Row         string                    `json:"row"`
	Destructive bool                      `json:"destructive,omitempty"`
	Error       string                    `json:"error"`
}

func newProfileUnavailableCommand(spec profileUnavailableSpec, options Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:          spec.Use,
		Short:        spec.Short,
		Args:         spec.Args,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runProfileUnavailableCommand(cmd, spec, options)
		},
	}
	if spec.FlagSet != nil {
		spec.FlagSet(cmd)
	}
	return cmd
}

func runProfileUnavailableCommand(cmd *cobra.Command, spec profileUnavailableSpec, options Options) error {
	command := "gormes profile " + spec.Name
	message := command + " is classified in the Hermes CLI parity manifest but is still row-backed in Gormes"
	asJSON, _ := cmd.Flags().GetBool("json")
	if asJSON {
		body, err := json.MarshalIndent(profileUnavailableReportJSON{
			Build:       buildProvenance(options),
			Action:      "profile_command_unavailable",
			Command:     command,
			Status:      gormescli.RowBackedStatus,
			Row:         "Gormes profile command binding",
			Destructive: spec.Destructive,
			Error:       message,
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(body))
	}
	return gormescli.NewExitCodeError(2, fmt.Errorf("%s", message))
}

func profileUnavailableJSONFlag(cmd *cobra.Command) {
	cmd.Flags().Bool("json", false, "emit machine-readable JSON for the row-backed unavailable command")
}

func profileUnavailableDeleteFlags(cmd *cobra.Command) {
	cmd.Flags().BoolP("yes", "y", false, "skip confirmation")
	profileUnavailableJSONFlag(cmd)
}

func profileUnavailableAliasFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("remove", false, "remove the wrapper alias")
	cmd.Flags().String("name", "", "custom alias name")
	profileUnavailableJSONFlag(cmd)
}

func profileUnavailableExportFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("output", "o", "", "output archive path")
	profileUnavailableJSONFlag(cmd)
}

func profileUnavailableImportFlags(cmd *cobra.Command) {
	cmd.Flags().String("name", "", "profile name")
	profileUnavailableJSONFlag(cmd)
}

func profileUnavailableInstallFlags(cmd *cobra.Command) {
	cmd.Flags().String("name", "", "profile name")
	cmd.Flags().Bool("alias", false, "create a shell wrapper alias")
	cmd.Flags().Bool("force", false, "overwrite an existing profile")
	cmd.Flags().BoolP("yes", "y", false, "skip confirmation")
	profileUnavailableJSONFlag(cmd)
}

func profileUnavailableUpdateFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("force-config", false, "overwrite config files")
	cmd.Flags().BoolP("yes", "y", false, "skip confirmation")
	profileUnavailableJSONFlag(cmd)
}

func newProfileInfoCommand(seams Seams, options Options) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "info <name>",
		Short:        "Show a profile distribution manifest",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfileInfoCommand(cmd, seams, args[0], asJSON, options)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit a `{build, name, root, distribution}` JSON document with the profile root redacted")
	return cmd
}

func newProfileListCommand(seams Seams, options Options) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List known Gormes profiles and mark the active one",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runProfileListCommand(cmd, seams, asJSON, options)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit a `{build, active, profiles: [...]}` JSON document (suitable for fleet inventory automation)")
	return cmd
}

func runProfileShowCommand(cmd *cobra.Command, seams Seams, asJSON bool, options Options) error {
	selector := newProfileSelectorFromSeams(seams)
	profile, err := selector.Select(context.Background())
	if err != nil {
		if errors.Is(err, cli.ErrSelectorNoMatch) {
			if asJSON {
				return emitProfileShowJSON(cmd, "", "", nil, options)
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
		return emitProfileShowJSON(cmd, profile.Name, redactProfileRootPath(profile.RootPath), manifestPointer(manifest, hasManifest), options)
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
	Build        gormescli.BuildProvenance       `json:"build"`
	Active       string                          `json:"active"`
	Root         string                          `json:"root"`
	Distribution *profileDistributionSummaryJSON `json:"distribution,omitempty"`
}

func emitProfileShowJSON(cmd *cobra.Command, active, root string, manifest *cli.ProfileDistributionManifest, options Options) error {
	body, err := json.MarshalIndent(profileShowReportJSON{
		Build:        buildProvenance(options),
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

func runProfileSetCommand(cmd *cobra.Command, seams Seams, rawName string, action string, options Options) error {
	name := strings.TrimSpace(rawName)
	verb := strings.TrimSpace(action)
	if verb == "" {
		verb = "use"
	}
	if seams.ValidateProfileName == nil || seams.WriteActiveProfile == nil || seams.ResolveProfileRoot == nil {
		return fmt.Errorf("gormes profile %s: %w", verb, cli.ErrSelectorHelperUnavailable)
	}
	if err := seams.ValidateProfileName(name); err != nil {
		return fmt.Errorf("gormes profile %s %q: %w: %w", verb, name, errProfileNameInvalid, err)
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
				return fmt.Errorf("gormes profile %s %q: unknown profile (known: %s)",
					verb, name, strings.Join(known, ", "))
			}
		}
	}
	if err := seams.WriteActiveProfile(name); err != nil {
		return fmt.Errorf("gormes profile %s %q: %w: %w", verb, name, errProfileRootUnwritable, err)
	}
	root, err := seams.ResolveProfileRoot(name)
	if err != nil {
		return fmt.Errorf("gormes profile %s %q: %w: %w", verb, name, errProfileSetPartialFailure, err)
	}
	asJSON, _ := cmd.Flags().GetBool("json")
	if asJSON {
		body, marshalErr := json.MarshalIndent(profileSetReportJSON{
			Build:  buildProvenance(options),
			Action: verb,
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

func runProfileCreateCommand(cmd *cobra.Command, seams Seams, rawName string, cloneAll bool, asJSON bool, options Options) error {
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
			Build:    buildProvenance(options),
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

func runProfileListCommand(cmd *cobra.Command, seams Seams, asJSON bool, options Options) error {
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
			Build:    buildProvenance(options),
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

func runProfileInfoCommand(cmd *cobra.Command, seams Seams, rawName string, asJSON bool, options Options) error {
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
		return emitProfileInfoJSON(cmd, name, redactProfileRootPath(root), manifestPointer(manifest, ok), options)
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
	Build    gormescli.BuildProvenance `json:"build"`
	Active   string                    `json:"active"`
	Profiles []profileListEntryJSON    `json:"profiles"`
}

type profileListEntryJSON struct {
	Name         string                          `json:"name"`
	Active       bool                            `json:"active"`
	Distribution *profileDistributionSummaryJSON `json:"distribution,omitempty"`
}

type profileInfoReportJSON struct {
	Build        gormescli.BuildProvenance        `json:"build"`
	Name         string                           `json:"name"`
	Root         string                           `json:"root"`
	Distribution *cli.ProfileDistributionManifest `json:"distribution,omitempty"`
}

func emitProfileInfoJSON(cmd *cobra.Command, name, root string, manifest *cli.ProfileDistributionManifest, options Options) error {
	body, err := json.MarshalIndent(profileInfoReportJSON{
		Build:        buildProvenance(options),
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

func readProfileDistributionForName(seams Seams, name string) (cli.ProfileDistributionManifest, bool, error) {
	if seams.ResolveProfileRoot == nil || seams.ReadDistributionManifest == nil {
		return cli.ProfileDistributionManifest{}, false, nil
	}
	root, err := seams.ResolveProfileRoot(name)
	if err != nil {
		return cli.ProfileDistributionManifest{}, false, err
	}
	return readProfileDistributionForRoot(seams, root)
}

func readProfileDistributionForRoot(seams Seams, root string) (cli.ProfileDistributionManifest, bool, error) {
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
func newProfileSelectorFromSeams(seams Seams) cli.ProfileSelector {
	return cli.NewDefaultProfileSelector(cli.DefaultProfileSelectorOptions{
		ReadActiveProfileName: seams.ReadActiveProfileName,
		ValidateProfileName:   seams.ValidateProfileName,
		ResolveProfileRoot:    seams.ResolveProfileRoot,
	})
}

// DefaultSeams wires the production helpers from internal/cli
// into Seams. The active-profile file lives at
// GormesHome()/active_profile so the binding inherits whatever GORMES_HOME
// override is in effect for the process. Tests skip this and inject fakes via
// NewCommandWithSeams.
func DefaultSeams() Seams {
	activePath := filepath.Join(config.GormesHome(), "active_profile")
	return Seams{
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
			return DefaultListKnownProfiles()
		},
		ReadDistributionManifest: cli.ReadProfileDistributionManifest,
	}
}

// DefaultListKnownProfiles enumerates known profiles by reading the on-disk
// layout that ResolveProfileRoot produces. The default profile is always
// reported even if no profile dir exists yet so operators can always orient.
func DefaultListKnownProfiles() ([]string, error) {
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
