package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
)

type updateCommandSeams struct {
	CheckoutDir  func() (string, error)
	RunLifecycle func(context.Context, cli.UpdateLifecycleOptions) cli.UpdateReport
	// SkillSyncFor builds a SkillSyncRunner closure for a given checkout
	// directory. Override in tests to inject a fake; the default builds
	// the real adapter that calls internal/skills.SyncBundledSkillsToProfiles
	// against `<checkoutDir>/skills` and the active profile root.
	SkillSyncFor func(checkoutDir string) cli.SkillSyncRunner
}

func newUpdateCommand() *cobra.Command {
	return newUpdateCommandWithSeams(updateCommandSeams{})
}

func newUpdateCommandWithSeams(seams updateCommandSeams) *cobra.Command {
	var branch string
	var checkOnly bool
	var yes bool
	var restartGateway string
	var killStaleDashboard bool
	var backup bool
	var noBackup bool

	if seams.CheckoutDir == nil {
		seams.CheckoutDir = os.Getwd
	}
	if seams.RunLifecycle == nil {
		seams.RunLifecycle = cli.RunUpdateLifecycle
	}
	if seams.SkillSyncFor == nil {
		seams.SkillSyncFor = defaultSkillSyncFor
	}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a managed Gormes source checkout",
		RunE: func(cmd *cobra.Command, _ []string) error {
			checkoutDir, err := seams.CheckoutDir()
			if err != nil {
				return err
			}
			report := seams.RunLifecycle(cmd.Context(), cli.UpdateLifecycleOptions{
				CheckoutDir:        checkoutDir,
				Branch:             branch,
				CheckOnly:          checkOnly,
				Yes:                yes,
				RestartGateway:     restartGateway,
				KillStaleDashboard: killStaleDashboard,
				Backup:             backup,
				NoBackup:           noBackup,
				SkillSync:          seams.SkillSyncFor(checkoutDir),
				Git:                cli.RealUpdateGitRunner{},
			})
			if report.Branch == "" {
				report.Branch = branch
			}
			if checkOnly && !updateReportHasEvidence(report, cli.UpdateEvidenceCheck) {
				report.Evidence = append([]cli.UpdateEvidence{{Kind: cli.UpdateEvidenceCheck, Detail: "no checkout mutations requested"}}, report.Evidence...)
			}
			printUpdateReport(cmd, report)
			if report.Failed {
				message := "gormes update failed"
				if report.OperatorRecovery != "" {
					message += ": " + report.OperatorRecovery
				}
				return newExitCodeError(1, fmt.Errorf("%s", message))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&branch, "branch", "main", "Git branch to fetch and fast-forward")
	cmd.Flags().BoolVar(&checkOnly, "check", false, "check update readiness without mutating the checkout")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "assume yes for non-destructive recovery prompts")
	cmd.Flags().StringVar(&restartGateway, "restart-gateway", "auto", "restart policy for a live gateway: auto, always, or never")
	cmd.Flags().BoolVar(&killStaleDashboard, "kill-stale-dashboard", false, "stop stale dashboard processes after a successful update")
	cmd.Flags().BoolVar(&backup, "backup", false, "create a single-run pre-update backup of ~/.gormes (writer is a follow-up slice; this surface emits the policy decision)")
	cmd.Flags().BoolVar(&noBackup, "no-backup", false, "force-skip the pre-update backup; beats --backup and config opt-in")
	return cmd
}

func updateReportHasEvidence(report cli.UpdateReport, kind cli.UpdateEvidenceKind) bool {
	for _, evidence := range report.Evidence {
		if evidence.Kind == kind {
			return true
		}
	}
	return false
}

func printUpdateReport(cmd *cobra.Command, report cli.UpdateReport) {
	out := cmd.OutOrStdout()
	branch := report.Branch
	if branch == "" {
		branch = "main"
	}
	fmt.Fprintln(out, cli.Bold(out, "⚕ Updating Gormes Agent..."))
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s %s\n", cli.Dim(out, "update branch:"), branch)
	if report.PreviousBranch != "" {
		fmt.Fprintf(out, "%s %s\n", cli.Dim(out, "previous branch:"), report.PreviousBranch)
	}
	for _, evidence := range report.Evidence {
		glyph, color := updateGlyphAndColor(out, evidence.Kind)
		kind := color(out, string(evidence.Kind))
		detail := strings.TrimSpace(evidence.Detail)
		if detail == "" {
			fmt.Fprintf(out, "%s %s\n", glyph, kind)
			continue
		}
		fmt.Fprintf(out, "%s %s\t%s\n", glyph, kind, detail)
	}
	fmt.Fprintln(out)
	if report.Failed {
		fmt.Fprintln(out, cli.Bold(out, "✗ Update failed"))
	} else {
		fmt.Fprintln(out, cli.Green(out, cli.Bold(out, "✓ Update complete!")))
	}
	if report.OperatorRecovery != "" {
		fmt.Fprintln(cmd.ErrOrStderr(), report.OperatorRecovery)
	}
}

// defaultSkillSyncFor builds the production SkillSyncRunner: scan
// `<checkoutDir>/skills` for bundled SKILL.md files and sync them into the
// active profile root (config.GormesHome). Returns nil when the bundled
// root or profile root is missing — callers treat nil as "no sync to do"
// and emit no evidence (silent default).
//
// Adapter responsibility: convert internal/skills.BundledSkillProfileSyncReport
// counts to the import-free cli.SkillSyncResult shape so internal/cli has
// no skills-package dependency.
func defaultSkillSyncFor(checkoutDir string) cli.SkillSyncRunner {
	bundledRoot := filepath.Join(checkoutDir, "skills")
	if info, err := os.Stat(bundledRoot); err != nil || !info.IsDir() {
		return nil
	}
	profileRoot := config.GormesHome()
	if profileRoot == "" {
		return nil
	}
	return func(ctx context.Context) (cli.SkillSyncResult, error) {
		report, err := skills.SyncBundledSkillsToProfiles(ctx, skills.BundledSkillProfileSyncRequest{
			BundledRoot: bundledRoot,
			Profiles: []skills.SkillProfileRoot{
				{Name: "default", Root: profileRoot},
			},
		})
		if err != nil {
			return cli.SkillSyncResult{}, err
		}
		out := cli.SkillSyncResult{Profiles: make([]cli.SkillSyncProfileResult, 0, len(report.Summaries))}
		for _, s := range report.Summaries {
			out.Profiles = append(out.Profiles, cli.SkillSyncProfileResult{
				Profile:   s.Profile,
				Added:     s.Added,
				Unchanged: s.Unchanged,
				Conflicts: s.Conflicts,
				Failed:    s.Failed,
			})
		}
		return out, nil
	}
}

// updateGlyphAndColor maps an UpdateEvidenceKind to a status glyph and the
// color helper used to wrap the kind string. Mapping is by name suffix so
// adding a new evidence kind doesn't require editing this function unless
// its semantic class differs from the suffix convention.
//
//	*_failed, *_error               → ✗ (bold)
//	*_unavailable, *_timeout        → ⚠ (yellow)
//	update_check, *_log_mirrored,
//	  update_not_managed_checkout   → ℹ (dim)
//	default                         → ✓ (green)
func updateGlyphAndColor(w io.Writer, kind cli.UpdateEvidenceKind) (string, func(io.Writer, string) string) {
	s := string(kind)
	switch {
	case strings.HasSuffix(s, "_failed"), strings.HasSuffix(s, "_error"):
		return cli.Bold(w, "✗"), cli.Bold
	case strings.HasSuffix(s, "_unavailable"), strings.HasSuffix(s, "_timeout"):
		return cli.Yellow(w, "⚠"), cli.Yellow
	case strings.HasSuffix(s, "_skipped"), s == "update_check", strings.HasSuffix(s, "_log_mirrored"), s == "update_not_managed_checkout":
		return cli.Dim(w, "ℹ"), cli.Dim
	case strings.HasSuffix(s, "_requested"):
		return cli.BrightCyan(w, "◆"), cli.BrightCyan
	default:
		return cli.Green(w, "✓"), cli.Green
	}
}
