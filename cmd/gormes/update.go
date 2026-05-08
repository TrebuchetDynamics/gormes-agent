package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	// WebBuildFor builds a WebBuildRunner closure for the given checkout
	// directory and skip flag. Override in tests; default builds the real
	// adapter that runs npm install + npm run build in `<checkoutDir>/web`.
	WebBuildFor func(checkoutDir string, skipWeb bool) cli.WebBuildRunner
	// ConfigCheckFn returns the on-disk config version vs. latest. Override
	// in tests; default wraps internal/config.Check.
	ConfigCheckFn cli.ConfigCheckRunner
	// ConfigMigrateFn applies the latest schema migrations. Override in
	// tests; default wraps internal/config.MigrateConfigFile against the
	// resolved config path.
	ConfigMigrateFn cli.ConfigMigrateRunner
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
	var skipWeb bool

	if seams.CheckoutDir == nil {
		seams.CheckoutDir = os.Getwd
	}
	if seams.RunLifecycle == nil {
		seams.RunLifecycle = cli.RunUpdateLifecycle
	}
	if seams.SkillSyncFor == nil {
		seams.SkillSyncFor = defaultSkillSyncFor
	}
	if seams.WebBuildFor == nil {
		seams.WebBuildFor = defaultWebBuildFor
	}
	if seams.ConfigCheckFn == nil {
		seams.ConfigCheckFn = defaultConfigCheck
	}
	if seams.ConfigMigrateFn == nil {
		seams.ConfigMigrateFn = defaultConfigMigrate
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
				WebBuild:           seams.WebBuildFor(checkoutDir, skipWeb),
				ConfigCheck:        seams.ConfigCheckFn,
				ConfigMigrate:      seams.ConfigMigrateFn,
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
	cmd.Flags().BoolVar(&skipWeb, "skip-web", false, "skip the web UI rebuild step after pulling source")
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

// defaultConfigCheck wraps internal/config.Check into a context-taking
// closure suitable for the lifecycle's ConfigCheckRunner seam.
func defaultConfigCheck(_ context.Context) (cli.ConfigVersionResult, error) {
	report, err := config.Check()
	if err != nil {
		return cli.ConfigVersionResult{}, err
	}
	return cli.ConfigVersionResult{Current: report.ConfigVersion, Latest: report.LatestVersion}, nil
}

// defaultConfigMigrate wraps internal/config.MigrateConfigFile against
// the operator's current config path.
func defaultConfigMigrate(_ context.Context) error {
	_, err := config.MigrateConfigFile(config.ConfigPath())
	return err
}

// defaultWebBuildFor builds the production WebBuildRunner. Behavior:
//
//	skipWeb=true                            → runner returns Skipped
//	`<checkoutDir>/web/package.json` absent → factory returns nil (silent default)
//	npm not on PATH                         → runner returns Unavailable
//	otherwise                               → runner runs `npm install --silent`
//	                                          then `npm run build` in the web/
//	                                          dir; non-zero exit returns error
func defaultWebBuildFor(checkoutDir string, skipWeb bool) cli.WebBuildRunner {
	webDir := filepath.Join(checkoutDir, "web")
	pkgJSON := filepath.Join(webDir, "package.json")
	if _, err := os.Stat(pkgJSON); err != nil {
		// No web/ tree means there's nothing to rebuild — silent default.
		return nil
	}
	return func(ctx context.Context) (cli.WebBuildResult, error) {
		if skipWeb {
			return cli.WebBuildResult{Skipped: true, Reason: "--skip-web flag"}, nil
		}
		if _, err := exec.LookPath("npm"); err != nil {
			return cli.WebBuildResult{Unavailable: true, Reason: "npm not on PATH; install Node.js to enable web UI rebuild"}, nil
		}
		install := exec.CommandContext(ctx, "npm", "install", "--silent")
		install.Dir = webDir
		if out, err := install.CombinedOutput(); err != nil {
			return cli.WebBuildResult{}, fmt.Errorf("npm install failed: %v: %s", err, strings.TrimSpace(string(out)))
		}
		build := exec.CommandContext(ctx, "npm", "run", "build")
		build.Dir = webDir
		if out, err := build.CombinedOutput(); err != nil {
			return cli.WebBuildResult{}, fmt.Errorf("npm run build failed: %v: %s", err, strings.TrimSpace(string(out)))
		}
		return cli.WebBuildResult{Detail: "web UI built in " + webDir}, nil
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
	case strings.HasSuffix(s, "_unavailable"), strings.HasSuffix(s, "_timeout"), strings.HasSuffix(s, "_needed"):
		return cli.Yellow(w, "⚠"), cli.Yellow
	case strings.HasSuffix(s, "_skipped"), s == "update_check", strings.HasSuffix(s, "_log_mirrored"), s == "update_not_managed_checkout":
		return cli.Dim(w, "ℹ"), cli.Dim
	case strings.HasSuffix(s, "_requested"):
		return cli.BrightCyan(w, "◆"), cli.BrightCyan
	default:
		return cli.Green(w, "✓"), cli.Green
	}
}
