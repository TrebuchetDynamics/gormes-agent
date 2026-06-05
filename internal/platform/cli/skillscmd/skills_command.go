package skillscmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"text/tabwriter"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/spf13/cobra"
)

// SkillsURLInstallDeps wires the policy seams used by `gormes skills install
// <https://.../SKILL.md>`. Tests inject fakes here so no live HTTP, scan, or
// real filesystem mutation occurs.
type SkillsURLInstallDeps struct {
	Fetcher skills.URLFetcher
	Scanner skills.QuarantineScanner
	Store   skills.SkillStore
	Console skills.InteractiveConsole
}

type SkillsCommandDeps struct {
	ListInstalledSkills func(skills.ListOptions, map[string]struct{}) []skills.SkillRow
	DisabledSkills      func(platform string) map[string]struct{}
	URLInstall          SkillsURLInstallDeps
	// BuildProvenance returns the binary attribution payload included
	// at the top of `--json` output. Optional: when nil, the JSON
	// document omits the `build` field. cmd/gormes injects
	// newBuildProvenance() so fleet automation can attribute the
	// inventory to the binary version that emitted it.
	BuildProvenance func() any
}

func NewSkillsCommand(deps SkillsCommandDeps) *cobra.Command {
	root := &cobra.Command{
		Use:   "skills",
		Short: "Manage skills",
		Args:  cobra.NoArgs,
	}
	root.AddCommand(NewSkillsListCommand(deps))
	root.AddCommand(NewSkillsInstallCommand(deps))
	return root
}

// NewSkillsInstallCommand binds the URL install policy from internal/skills
// to a cobra subcommand. The `<identifier>` is currently scoped to direct
// HTTPS SKILL.md URLs; other source adapters are out of scope for this row.
func NewSkillsInstallCommand(deps SkillsCommandDeps) *cobra.Command {
	var nameOverride, categoryOverride string
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "install <url>",
		Short: "Install a skill from a direct SKILL.md URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := skills.URLInstallRequest{
				URL:              args[0],
				NameOverride:     nameOverride,
				CategoryOverride: categoryOverride,
				Interactive:      false,
			}
			policy := skills.URLInstallPolicy{
				Fetcher: deps.URLInstall.Fetcher,
				Scanner: deps.URLInstall.Scanner,
				Store:   deps.URLInstall.Store,
				Console: deps.URLInstall.Console,
			}
			ev := skills.PerformURLInstall(cmd.Context(), policy, req)
			switch ev.Code {
			case "url_skill_installed":
				if asJSON {
					return writeSkillsInstallJSON(cmd, ev, deps.BuildProvenance)
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "installed %s\n", ev.InstalledPath)
				return nil
			default:
				if ev.Reason != "" {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), ev.Reason)
				}
				return errors.New(ev.Code)
			}
		},
	}
	cmd.Flags().StringVar(&nameOverride, "name", "", "explicit skill name override (required for non-interactive URL installs without a safe resolved name)")
	cmd.Flags().StringVar(&categoryOverride, "category", "", "optional category bucket under the active store")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: {build, action, name, installed_path}")
	return cmd
}

// skillsInstallReportJSON is the wire shape for `skills install --json`.
// Fleet automation rolling out skills across machines parses this to
// confirm where each install landed without scraping prose. The
// installed path is the reachable file the URL install wrote.
type skillsInstallReportJSON struct {
	Build         any    `json:"build,omitempty"`
	Action        string `json:"action"`
	InstalledPath string `json:"installed_path"`
}

func writeSkillsInstallJSON(cmd *cobra.Command, ev skills.URLInstallEvidence, buildProv func() any) error {
	report := skillsInstallReportJSON{
		Action:        "installed",
		InstalledPath: ev.InstalledPath,
	}
	if buildProv != nil {
		report.Build = buildProv()
	}
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(body))
	return err
}

func NewSkillsListCommand(deps SkillsCommandDeps) *cobra.Command {
	if deps.ListInstalledSkills == nil {
		deps.ListInstalledSkills = skills.ListInstalledSkills
	}
	if deps.DisabledSkills == nil {
		deps.DisabledSkills = func(string) map[string]struct{} { return nil }
	}

	var opts skills.ListOptions
	var asJSON bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List installed skills",
		RunE: func(cmd *cobra.Command, _ []string) error {
			source := normalizedSkillsListSource(opts.Source)
			if source == "" {
				return fmt.Errorf("invalid skills list source %q", opts.Source)
			}
			opts.Source = source
			disabled := deps.DisabledSkills("")
			rows := deps.ListInstalledSkills(opts, disabled)
			if asJSON {
				return writeSkillsListJSON(cmd, rows, opts, deps.BuildProvenance)
			}
			return writeSkillsList(cmd, rows, opts.EnabledOnly)
		},
	}
	cmd.Flags().StringVar(&opts.Source, "source", "all", "filter by installed skill source: all, hub, builtin, local, or external")
	cmd.Flags().BoolVar(&opts.EnabledOnly, "enabled-only", false, "hide disabled skills")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: {build, source, enabled_only, counts, skills}")
	return cmd
}

type skillsListReportJSON struct {
	Build       any                   `json:"build,omitempty"`
	Source      string                `json:"source"`
	EnabledOnly bool                  `json:"enabled_only"`
	Counts      skillsListCountsJSON  `json:"counts"`
	Skills      []skillsListEntryJSON `json:"skills"`
}

type skillsListCountsJSON struct {
	Enabled  int `json:"enabled"`
	Disabled int `json:"disabled"`
}

type skillsListEntryJSON struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Source   string `json:"source"`
	Trust    string `json:"trust"`
	Status   string `json:"status"`
	Path     string `json:"path,omitempty"`
}

func writeSkillsListJSON(cmd *cobra.Command, rows []skills.SkillRow, opts skills.ListOptions, buildProv func() any) error {
	report := skillsListReportJSON{
		Source:      opts.Source,
		EnabledOnly: opts.EnabledOnly,
		Skills:      make([]skillsListEntryJSON, 0, len(rows)),
	}
	if buildProv != nil {
		report.Build = buildProv()
	}
	for _, row := range rows {
		if row.Status == skills.SkillStatusDisabled {
			report.Counts.Disabled++
		} else {
			report.Counts.Enabled++
		}
		report.Skills = append(report.Skills, skillsListEntryJSON{
			Name:     row.Name,
			Category: row.Category,
			Source:   row.Source,
			Trust:    row.Trust,
			Status:   string(row.Status),
			Path:     row.Path,
		})
	}
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(body))
	return err
}

func writeSkillsList(cmd *cobra.Command, rows []skills.SkillRow, enabledOnly bool) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "Name\tCategory\tSource\tTrust\tStatus"); err != nil {
		return err
	}

	enabledCount := 0
	disabledCount := 0
	for _, row := range rows {
		if row.Status == skills.SkillStatusDisabled {
			disabledCount++
		} else {
			enabledCount++
		}
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", row.Name, row.Category, row.Source, row.Trust, row.Status); err != nil {
			return err
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}

	if enabledOnly {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "%d enabled shown\n", enabledCount)
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "%d enabled, %d disabled\n", enabledCount, disabledCount)
	return err
}

func normalizedSkillsListSource(source string) string {
	switch source {
	case "", "all":
		return "all"
	case "hub", "builtin", "local", "external":
		return source
	default:
		return ""
	}
}
