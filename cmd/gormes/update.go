package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
	// BackupWriterFor builds a BackupWriter closure for the active gormes
	// home. Override in tests; default writes a zip of GormesHome (with
	// IsExcludedFromBackup paths skipped) to
	// `<home>/backups/pre-update-<UTC>.zip`.
	BackupWriterFor func() cli.BackupWriter
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
	var asJSON bool

	if seams.CheckoutDir == nil {
		seams.CheckoutDir = resolveManagedCheckoutDir
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
	if seams.BackupWriterFor == nil {
		seams.BackupWriterFor = defaultBackupWriterFor
	}

	cmd := &cobra.Command{
		Use:          "update",
		Short:        "Update a managed Gormes source checkout",
		SilenceUsage: true,
		Long: `Update a managed Gormes source checkout: fetch + fast-forward, sync
bundled skills, rebuild the web UI, run a config schema migration check,
and restart the gateway.

Backup and rollback:

  gormes update --backup           take a pre-update zip of GORMES_HOME
                                   before pulling source. Set
                                   ` + "`[updates] pre_update_backup = true`" + ` in
                                   config.toml to make this the default.
  gormes restore --list            enumerate available pre-update zips,
                                   newest first.
  gormes restore --latest --yes    roll back to the most recent zip
                                   (overwrites files in GORMES_HOME).

When ` + "`--backup`" + ` is set and the update later fails, the report ends with
a ` + "`◆ update_rollback_hint`" + ` line spelling out the restore command above
so the recovery path is visible inline.
`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			checkoutDir, err := seams.CheckoutDir()
			if err != nil {
				return err
			}
			// Best-effort config read: when config.Load fails or no
			// config exists, fall through with zero-value Updates so
			// `gormes update` keeps working on a fresh install with
			// no config.toml.
			var updatesCfg config.UpdatesCfg
			if cfg, cfgErr := config.Load(nil); cfgErr == nil {
				updatesCfg = cfg.Updates
			}
			report := seams.RunLifecycle(cmd.Context(), cli.UpdateLifecycleOptions{
				CheckoutDir:         checkoutDir,
				Branch:              branch,
				CheckOnly:           checkOnly,
				Yes:                 yes,
				RestartGateway:      restartGateway,
				KillStaleDashboard:  killStaleDashboard,
				Backup:              backup,
				NoBackup:            noBackup,
				BackupConfigEnabled: updatesCfg.PreUpdateBackup,
				SkillSync:           seams.SkillSyncFor(checkoutDir),
				WebBuild:            seams.WebBuildFor(checkoutDir, skipWeb),
				ConfigCheck:         seams.ConfigCheckFn,
				ConfigMigrate:       seams.ConfigMigrateFn,
				BackupWriter:        seams.BackupWriterFor(),
				Git:                 cli.RealUpdateGitRunner{},
			})
			if report.Branch == "" {
				report.Branch = branch
			}
			if checkOnly && !updateReportHasEvidence(report, cli.UpdateEvidenceCheck) {
				report.Evidence = append([]cli.UpdateEvidence{{Kind: cli.UpdateEvidenceCheck, Detail: "no checkout mutations requested"}}, report.Evidence...)
			}
			if asJSON {
				if err := printUpdateReportJSON(cmd, report); err != nil {
					return err
				}
			} else {
				printUpdateReport(cmd, report, checkOnly)
				if !report.Failed {
					printCuratorRecentRunNotice(cmd)
				}
			}
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
	cmd.Flags().BoolVar(&backup, "backup", false, "create a single-run pre-update backup zip of ~/.gormes; restore later with `gormes restore --latest --yes`")
	cmd.Flags().BoolVar(&noBackup, "no-backup", false, "force-skip the pre-update backup; beats --backup and config opt-in")
	cmd.Flags().BoolVar(&skipWeb, "skip-web", false, "skip the web UI rebuild step after pulling source")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit a machine-readable JSON report instead of the human-readable progress UX (suitable for CI/cron consumers)")
	return cmd
}

// updateReportJSON shapes UpdateReport for machine-readable output.
// Mirrors the internal/cli struct field-for-field with snake_case JSON
// tags. Defined in cmd/gormes so internal/cli stays free of presentation
// concerns.
type updateReportJSON struct {
	Build            buildProvenanceJSON  `json:"build"`
	Branch           string               `json:"branch"`
	PreviousBranch   string               `json:"previous_branch,omitempty"`
	Failed           bool                 `json:"failed"`
	Evidence         []updateEvidenceJSON `json:"evidence"`
	OperatorRecovery string               `json:"operator_recovery,omitempty"`
	DashboardPIDs    []int                `json:"dashboard_pids,omitempty"`
}

type updateEvidenceJSON struct {
	Kind   string `json:"kind"`
	Detail string `json:"detail,omitempty"`
}

func printUpdateReportJSON(cmd *cobra.Command, report cli.UpdateReport) error {
	out := cmd.OutOrStdout()
	shaped := updateReportJSON{
		Build:            newBuildProvenance(),
		Branch:           report.Branch,
		PreviousBranch:   report.PreviousBranch,
		Failed:           report.Failed,
		Evidence:         make([]updateEvidenceJSON, len(report.Evidence)),
		OperatorRecovery: report.OperatorRecovery,
		DashboardPIDs:    report.DashboardPIDs,
	}
	for i, e := range report.Evidence {
		shaped.Evidence[i] = updateEvidenceJSON{Kind: string(e.Kind), Detail: e.Detail}
	}
	body, err := json.MarshalIndent(shaped, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(out, string(body))
	return nil
}

func updateReportHasEvidence(report cli.UpdateReport, kind cli.UpdateEvidenceKind) bool {
	for _, evidence := range report.Evidence {
		if evidence.Kind == kind {
			return true
		}
	}
	return false
}

func printUpdateReport(cmd *cobra.Command, report cli.UpdateReport, checkOnly bool) {
	out := cmd.OutOrStdout()
	branch := report.Branch
	if branch == "" {
		branch = "main"
	}
	// `--check` is a readiness probe; the banner and summary must say
	// "Checking" and "Update check complete" so operators don't read
	// "Updating..." and "Update complete!" as evidence that an update
	// actually happened (it didn't).
	bannerText := "⚕ Updating Gormes Agent..."
	successText := "✓ Update complete!"
	if checkOnly {
		bannerText = "⚕ Checking Gormes Agent..."
		successText = "✓ Update check complete"
	}
	fmt.Fprintln(out, cli.Bold(out, bannerText))
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
		fmt.Fprintln(out, cli.Green(out, cli.Bold(out, successText)))
	}
	if report.OperatorRecovery != "" {
		fmt.Fprintln(cmd.ErrOrStderr(), report.OperatorRecovery)
	}
}

func printCuratorRecentRunNotice(cmd *cobra.Command) {
	root := resolveCuratorSkillsRoot(curatorCommandDeps{})
	curator := skills.NewCurator(skills.CuratorConfig{Root: root})
	state, err := curator.LoadState()
	if err != nil || state.LastRunAt == nil || state.LastRunAt.IsZero() {
		return
	}
	lastRun := state.LastRunAt.UTC()
	if state.LastRunSummaryShownAt != nil && state.LastRunSummaryShownAt.UTC().Equal(lastRun) {
		return
	}

	shownAt := lastRun
	state.LastRunSummaryShownAt = &shownAt
	summary := strings.TrimSpace(state.LastRunSummary)
	if !strings.Contains(summary, "\n") {
		_ = curator.SaveState(state)
		return
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "\nℹ Skill curator — last run %s\n", formatCuratorTimestamp(state.LastRunAt, curatorCommandDeps{}))
	for _, line := range strings.Split(summary, "\n") {
		fmt.Fprintf(out, "  %s\n", line)
	}
	fmt.Fprintln(out, "  (This message shows once per curator run. View anytime: gormes curator status)")
	_ = curator.SaveState(state)
}

// resolveManagedCheckoutDir is the production CheckoutDir resolver for
// `gormes update`. It mirrors install.sh's managed_checkout_dir():
//
//   - GORMES_INSTALL_DIR override wins outright;
//   - otherwise the install's managed clone lives at
//     `$GORMES_INSTALL_HOME/gormes-agent` (defaulting GORMES_INSTALL_HOME
//     to `$HOME/.gormes`).
//
// Critical safety property: this function NEVER falls back to
// `os.Getwd()`. A regression observed during a v0.2.0 fresh-install
// probe — `gormes update` walked up from the cwd to find the
// gormes-agent dev tree, switched its branch from `development` to
// `main`, and ran a web build there — directly traces to the previous
// `os.Getwd` default. The lifecycle's `update_not_managed_checkout`
// guard correctly fails the run when the resolved path is not a git
// worktree; honoring cwd would silently mutate the wrong tree.
func resolveManagedCheckoutDir() (string, error) {
	if dir := strings.TrimSpace(os.Getenv("GORMES_INSTALL_DIR")); dir != "" {
		return dir, nil
	}
	home := strings.TrimSpace(os.Getenv("GORMES_INSTALL_HOME"))
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve managed checkout: %w", err)
		}
		home = filepath.Join(userHome, ".gormes")
	}
	return filepath.Join(home, "gormes-agent"), nil
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

// defaultBackupKeep is the retention budget the production BackupWriter
// applies after each successful write when config.toml does not set
// `[updates] backup_keep`. Matches Hermes' default.
const defaultBackupKeep = 5

// resolveBackupKeep reads `[updates] backup_keep` from config and falls
// back to defaultBackupKeep when the value is missing, zero, or
// negative. The fallback for non-positive values matches PruneBackups'
// keep<=0 safety: an operator who set 0 by mistake should not lose
// every backup on the next run.
func resolveBackupKeep() int {
	cfg, err := config.Load(nil)
	if err != nil {
		return defaultBackupKeep
	}
	if cfg.Updates.BackupKeep <= 0 {
		return defaultBackupKeep
	}
	return cfg.Updates.BackupKeep
}

// defaultBackupWriterFor builds the production BackupWriter. Returns nil
// when GormesHome is unset (so the lifecycle keeps the policy-only
// behavior). Otherwise returns a closure that writes a UTC-stamped zip
// under `<home>/backups/pre-update-<UTC>.zip`, skipping the existing
// IsExcludedFromBackup paths (checkpoints/, backups/, *.db-{wal,shm,journal}),
// then prunes older backups in the same directory to keep the newest
// resolveBackupKeep() files.
func defaultBackupWriterFor() cli.BackupWriter {
	home := config.GormesHome()
	if home == "" {
		return nil
	}
	keep := resolveBackupKeep()
	return func(ctx context.Context) (cli.BackupResult, error) {
		stamp := time.Now().UTC().Format("20060102T150405Z")
		dest := filepath.Join(home, "backups", "pre-update-"+stamp+".zip")
		res, err := cli.WriteBackupZip(ctx, home, dest)
		if err != nil {
			return res, err
		}
		// Prune is best-effort: a prune failure must not surface as a
		// backup-write failure (the new zip is already on disk and
		// usable).
		if count, freed, _ := cli.PruneBackups(filepath.Dir(dest), keep); count > 0 {
			res.PrunedCount = count
			res.PrunedBytes = freed
		}
		return res, nil
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
	case strings.HasSuffix(s, "_requested"), strings.HasSuffix(s, "_hint"):
		return cli.BrightCyan(w, "◆"), cli.BrightCyan
	default:
		return cli.Green(w, "✓"), cli.Green
	}
}
