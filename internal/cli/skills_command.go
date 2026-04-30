package cli

import (
	"errors"
	"fmt"
	"text/tabwriter"

	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
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
}

func NewSkillsCommand(deps SkillsCommandDeps) *cobra.Command {
	root := &cobra.Command{
		Use:   "skills",
		Short: "Manage skills",
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
	return cmd
}

func NewSkillsListCommand(deps SkillsCommandDeps) *cobra.Command {
	if deps.ListInstalledSkills == nil {
		deps.ListInstalledSkills = skills.ListInstalledSkills
	}
	if deps.DisabledSkills == nil {
		deps.DisabledSkills = func(string) map[string]struct{} { return nil }
	}

	var opts skills.ListOptions
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
			return writeSkillsList(cmd, rows, opts.EnabledOnly)
		},
	}
	cmd.Flags().StringVar(&opts.Source, "source", "all", "filter by installed skill source: all, hub, builtin, or local")
	cmd.Flags().BoolVar(&opts.EnabledOnly, "enabled-only", false, "hide disabled skills")
	return cmd
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
	case "hub", "builtin", "local":
		return source
	default:
		return ""
	}
}
