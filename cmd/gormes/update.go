package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
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
	// BinaryPublisherFor builds the native build/publish closure for the
	// managed checkout. Override in tests so command plumbing can be
	// exercised without running `go build` or touching PATH binaries.
	BinaryPublisherFor func(checkoutDir string) cli.UpdateBinaryPublisher
	// GatewayRestartFor builds the validated restart closure used after a
	// successful binary publish. Override in tests to avoid touching live
	// runtime/service state.
	GatewayRestartFor func() cli.UpdateGatewayRestartRunner
	// DetectInstallKind classifies the active installation without mutating
	// binaries or source checkouts. It routes release installs through the
	// release planner for --check and all installs through the planner for
	// --dry-run.
	DetectInstallKind func() cli.UpdateInstallKind
	// RuntimePlatform returns GOOS/GOARCH for release asset planning.
	RuntimePlatform func() (string, string)
	// LoadReleaseMetadata resolves the target release for release-planner
	// check/dry-run modes.
	LoadReleaseMetadata func(context.Context, cli.UpdateReleaseChannel) (cli.UpdateReleaseMetadata, error)
	// BuildReleasePlan is the pure planner seam used by tests to pin command
	// behavior independently from release metadata IO.
	BuildReleasePlan func(cli.UpdateReleasePlanOptions) cli.UpdateReleasePlan
	// RunReleaseBinaryUpdate applies a verified release artifact to the
	// managed and published binary paths for release installs.
	RunReleaseBinaryUpdate func(context.Context, cli.UpdateReleaseBinaryOptions) cli.UpdateReleaseBinaryReport
	// LoadReleaseAssetManifest loads the verified release manifest and
	// payload root used for release-installed asset and skill sync.
	LoadReleaseAssetManifest func(context.Context, cli.UpdateReleasePlan) (cli.UpdateReleaseManifest, string, error)
	// ReleaseAssetRoot resolves the release-installed static asset root.
	ReleaseAssetRoot func() (string, error)
	// ReleaseSkillProfiles returns every profile that should receive bundled
	// skill updates for release installs.
	ReleaseSkillProfiles func() ([]skills.SkillProfileRoot, error)
	// RunReleaseAssetSkillSync applies verified release assets and skills.
	RunReleaseAssetSkillSync func(context.Context, cli.UpdateReleaseAssetSkillSyncOptions) cli.UpdateReleaseAssetSkillSyncReport
	// RunReleaseRollback restores a prior release binary snapshot.
	RunReleaseRollback func(context.Context, cli.UpdateReleaseRollbackOptions) cli.UpdateReleaseBinaryReport
	// RunReleaseServiceCoordination wraps release update mutations with the
	// global update lock and managed-service drain/stop/restart choreography.
	RunReleaseServiceCoordination func(context.Context, cli.UpdateServiceCoordinationOptions) cli.UpdateReleaseBinaryReport
	// ReleaseUpdateLock returns the global lock used for release mutations.
	ReleaseUpdateLock func() cli.UpdateLock
	// ReleaseManagedServices returns managed services that should be stopped
	// before mutating a release install and restarted afterward.
	ReleaseManagedServices func() []cli.UpdateManagedService
	// ReleaseUnmanagedSessions returns active unmanaged sessions that should
	// block release mutation unless --force is set.
	ReleaseUnmanagedSessions func(context.Context) []cli.UpdateUnmanagedSession
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
	var dryRun bool
	var channel string
	var force bool
	var rollbackSnapshot string

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
	if seams.BinaryPublisherFor == nil {
		seams.BinaryPublisherFor = defaultBinaryPublisherFor
	}
	if seams.GatewayRestartFor == nil {
		seams.GatewayRestartFor = defaultUpdateGatewayRestartFor
	}
	if seams.DetectInstallKind == nil {
		seams.DetectInstallKind = detectUpdateInstallKind
	}
	if seams.RuntimePlatform == nil {
		seams.RuntimePlatform = func() (string, string) { return runtime.GOOS, runtime.GOARCH }
	}
	if seams.LoadReleaseMetadata == nil {
		seams.LoadReleaseMetadata = defaultLoadReleaseMetadata
	}
	if seams.BuildReleasePlan == nil {
		seams.BuildReleasePlan = cli.BuildUpdateReleasePlan
	}
	if seams.RunReleaseBinaryUpdate == nil {
		seams.RunReleaseBinaryUpdate = cli.RunUpdateReleaseBinaryUpdate
	}
	if seams.LoadReleaseAssetManifest == nil {
		seams.LoadReleaseAssetManifest = defaultLoadReleaseAssetManifest
	}
	if seams.ReleaseAssetRoot == nil {
		seams.ReleaseAssetRoot = resolveReleaseAssetRoot
	}
	if seams.ReleaseSkillProfiles == nil {
		seams.ReleaseSkillProfiles = defaultSkillSyncProfiles
	}
	if seams.RunReleaseAssetSkillSync == nil {
		seams.RunReleaseAssetSkillSync = cli.RunUpdateReleaseAssetSkillSync
	}
	if seams.RunReleaseRollback == nil {
		seams.RunReleaseRollback = cli.RunUpdateReleaseRollback
	}
	if seams.RunReleaseServiceCoordination == nil {
		seams.RunReleaseServiceCoordination = cli.RunUpdateServiceCoordination
	}
	if seams.ReleaseUpdateLock == nil {
		seams.ReleaseUpdateLock = defaultReleaseUpdateLock
	}
	if seams.ReleaseManagedServices == nil {
		seams.ReleaseManagedServices = defaultReleaseManagedServices
	}
	if seams.ReleaseUnmanagedSessions == nil {
		seams.ReleaseUnmanagedSessions = defaultReleaseUnmanagedSessions
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
			if rollbackSnapshot != "" {
				snapshotRoot, err := resolveUpdateReleaseSnapshotRoot()
				if err != nil {
					return err
				}
				report := runUpdateReleaseWithServiceCoordination(cmd.Context(), updateReleaseBinaryModeOptions{
					Force:                    force,
					RunServiceCoordination:   seams.RunReleaseServiceCoordination,
					ReleaseUpdateLock:        seams.ReleaseUpdateLock,
					ReleaseManagedServices:   seams.ReleaseManagedServices,
					ReleaseUnmanagedSessions: seams.ReleaseUnmanagedSessions,
				}, func(ctx context.Context) cli.UpdateReleaseBinaryReport {
					return seams.RunReleaseRollback(ctx, cli.UpdateReleaseRollbackOptions{
						SnapshotID:   rollbackSnapshot,
						SnapshotRoot: snapshotRoot,
					})
				})
				if asJSON {
					if err := printUpdateReleaseBinaryReportJSON(cmd, "update_rollback", report); err != nil {
						return err
					}
				} else {
					printUpdateReleaseBinaryReport(cmd, "update_rollback", report)
				}
				if report.Failed {
					return newExitCodeError(1, fmt.Errorf("gormes update rollback failed"))
				}
				return nil
			}
			installKind := seams.DetectInstallKind()
			if dryRun || (checkOnly && installKind == cli.UpdateInstallKindRelease) {
				return runUpdateReleasePlannerMode(cmd, updateReleasePlannerModeOptions{
					Action:               updateReleasePlannerAction(dryRun),
					InstallKind:          installKind,
					Channel:              cli.UpdateReleaseChannel(channel),
					AsJSON:               asJSON,
					RuntimePlatform:      seams.RuntimePlatform,
					LoadReleaseMetadata:  seams.LoadReleaseMetadata,
					BuildReleasePlan:     seams.BuildReleasePlan,
					SnapshotPathResolver: resolvePlannedUpdateSnapshotPath,
				})
			}
			if installKind == cli.UpdateInstallKindRelease {
				return runUpdateReleaseBinaryMode(cmd, updateReleaseBinaryModeOptions{
					Channel:                  cli.UpdateReleaseChannel(channel),
					AsJSON:                   asJSON,
					Force:                    force,
					RuntimePlatform:          seams.RuntimePlatform,
					LoadReleaseMetadata:      seams.LoadReleaseMetadata,
					BuildReleasePlan:         seams.BuildReleasePlan,
					RunReleaseUpdate:         seams.RunReleaseBinaryUpdate,
					LoadAssetManifest:        seams.LoadReleaseAssetManifest,
					ReleaseAssetRoot:         seams.ReleaseAssetRoot,
					ReleaseSkillProfiles:     seams.ReleaseSkillProfiles,
					RunAssetSkillSync:        seams.RunReleaseAssetSkillSync,
					RunServiceCoordination:   seams.RunReleaseServiceCoordination,
					ReleaseUpdateLock:        seams.ReleaseUpdateLock,
					ReleaseManagedServices:   seams.ReleaseManagedServices,
					ReleaseUnmanagedSessions: seams.ReleaseUnmanagedSessions,
					SnapshotPathResolver:     resolvePlannedUpdateSnapshotPath,
				})
			}
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
				BinaryPublisher:     seams.BinaryPublisherFor(checkoutDir),
				GatewayRestart:      seams.GatewayRestartFor(),
				Git:                 cli.RealUpdateGitRunner{},
			})
			if report.Branch == "" {
				report.Branch = branch
			}
			if checkOnly && !updateReportHasEvidence(report, cli.UpdateEvidenceCheck) {
				report.Evidence = append([]cli.UpdateEvidence{{Kind: cli.UpdateEvidenceCheck, Detail: "no checkout mutations requested"}}, report.Evidence...)
			}
			if !checkOnly {
				closeLog := attachUpdateLogMirror(cmd, &report)
				defer closeLog()
				appendUpdateLedgerEvidence(&report, restartGateway)
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
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "plan the update path without mutating binaries, assets, services, config, sessions, memory, git checkouts, logs, or ledgers")
	cmd.Flags().StringVar(&channel, "channel", string(cli.UpdateReleaseChannelStable), "update channel for planner mode: stable or development")
	cmd.Flags().BoolVar(&force, "force", false, "allow supported release-update policy overrides; integrity, provenance, platform, and smoke-test failures are never bypassed")
	cmd.Flags().StringVar(&rollbackSnapshot, "rollback", "", "restore a previous release binary snapshot by snapshot id")
	return cmd
}

type updateReleasePlannerModeOptions struct {
	Action               string
	InstallKind          cli.UpdateInstallKind
	Channel              cli.UpdateReleaseChannel
	AsJSON               bool
	RuntimePlatform      func() (string, string)
	LoadReleaseMetadata  func(context.Context, cli.UpdateReleaseChannel) (cli.UpdateReleaseMetadata, error)
	BuildReleasePlan     func(cli.UpdateReleasePlanOptions) cli.UpdateReleasePlan
	SnapshotPathResolver func() (string, error)
}

func updateReleasePlannerAction(dryRun bool) string {
	if dryRun {
		return "update_dry_run"
	}
	return "update_check"
}

func runUpdateReleasePlannerMode(cmd *cobra.Command, opts updateReleasePlannerModeOptions) error {
	plan, channelBlocker := buildUpdateReleaseCommandPlan(cmd.Context(), updateReleaseCommandPlanOptions{
		InstallKind:          opts.InstallKind,
		Channel:              opts.Channel,
		RuntimePlatform:      opts.RuntimePlatform,
		LoadReleaseMetadata:  opts.LoadReleaseMetadata,
		BuildReleasePlan:     opts.BuildReleasePlan,
		SnapshotPathResolver: opts.SnapshotPathResolver,
	})
	if channelBlocker != nil {
		plan.Blockers = append([]cli.UpdateReleaseBlocker{*channelBlocker}, plan.Blockers...)
	}
	failed := len(plan.Blockers) > 0
	if opts.AsJSON {
		if err := printUpdateReleasePlanJSON(cmd, opts.Action, plan, failed); err != nil {
			return err
		}
	} else {
		printUpdateReleasePlan(cmd, opts.Action, plan, failed)
	}
	if failed {
		return newExitCodeError(1, fmt.Errorf("gormes update planner blocked"))
	}
	if opts.Action == "update_check" && plan.UpdateAvailable {
		return newExitCodeError(10, fmt.Errorf("gormes update available"))
	}
	return nil
}

type updateReleaseCommandPlanOptions struct {
	InstallKind          cli.UpdateInstallKind
	Channel              cli.UpdateReleaseChannel
	RuntimePlatform      func() (string, string)
	LoadReleaseMetadata  func(context.Context, cli.UpdateReleaseChannel) (cli.UpdateReleaseMetadata, error)
	BuildReleasePlan     func(cli.UpdateReleasePlanOptions) cli.UpdateReleasePlan
	SnapshotPathResolver func() (string, error)
}

func buildUpdateReleaseCommandPlan(ctx context.Context, opts updateReleaseCommandPlanOptions) (cli.UpdateReleasePlan, *cli.UpdateReleaseBlocker) {
	channel, channelBlocker := normalizeUpdateReleaseChannel(opts.Channel)
	target, metadataErr := cli.UpdateReleaseMetadata{}, error(nil)
	if channel == cli.UpdateReleaseChannelStable && opts.LoadReleaseMetadata != nil {
		target, metadataErr = opts.LoadReleaseMetadata(ctx, channel)
	}
	goos, goarch := runtime.GOOS, runtime.GOARCH
	if opts.RuntimePlatform != nil {
		goos, goarch = opts.RuntimePlatform()
	}
	snapshotPath := ""
	if opts.SnapshotPathResolver != nil {
		path, err := opts.SnapshotPathResolver()
		if err != nil {
			metadataErr = err
		} else {
			snapshotPath = path
		}
	}
	buildPlan := opts.BuildReleasePlan
	if buildPlan == nil {
		buildPlan = cli.BuildUpdateReleasePlan
	}
	plan := buildPlan(cli.UpdateReleasePlanOptions{
		InstallKind:   opts.InstallKind,
		Channel:       channel,
		Current:       cli.UpdateBuildIdentity{Version: Version, GitCommit: resolveGitCommit()},
		Target:        target,
		MetadataError: metadataErr,
		GOOS:          goos,
		GOARCH:        goarch,
		SnapshotPath:  snapshotPath,
	})
	return plan, channelBlocker
}

type updateReleaseBinaryModeOptions struct {
	Channel                  cli.UpdateReleaseChannel
	AsJSON                   bool
	Force                    bool
	RuntimePlatform          func() (string, string)
	LoadReleaseMetadata      func(context.Context, cli.UpdateReleaseChannel) (cli.UpdateReleaseMetadata, error)
	BuildReleasePlan         func(cli.UpdateReleasePlanOptions) cli.UpdateReleasePlan
	RunReleaseUpdate         func(context.Context, cli.UpdateReleaseBinaryOptions) cli.UpdateReleaseBinaryReport
	LoadAssetManifest        func(context.Context, cli.UpdateReleasePlan) (cli.UpdateReleaseManifest, string, error)
	ReleaseAssetRoot         func() (string, error)
	ReleaseSkillProfiles     func() ([]skills.SkillProfileRoot, error)
	RunAssetSkillSync        func(context.Context, cli.UpdateReleaseAssetSkillSyncOptions) cli.UpdateReleaseAssetSkillSyncReport
	RunServiceCoordination   func(context.Context, cli.UpdateServiceCoordinationOptions) cli.UpdateReleaseBinaryReport
	ReleaseUpdateLock        func() cli.UpdateLock
	ReleaseManagedServices   func() []cli.UpdateManagedService
	ReleaseUnmanagedSessions func(context.Context) []cli.UpdateUnmanagedSession
	SnapshotPathResolver     func() (string, error)
}

func runUpdateReleaseBinaryMode(cmd *cobra.Command, opts updateReleaseBinaryModeOptions) error {
	plan, channelBlocker := buildUpdateReleaseCommandPlan(cmd.Context(), updateReleaseCommandPlanOptions{
		InstallKind:          cli.UpdateInstallKindRelease,
		Channel:              opts.Channel,
		RuntimePlatform:      opts.RuntimePlatform,
		LoadReleaseMetadata:  opts.LoadReleaseMetadata,
		BuildReleasePlan:     opts.BuildReleasePlan,
		SnapshotPathResolver: opts.SnapshotPathResolver,
	})
	if channelBlocker != nil {
		plan.Blockers = append([]cli.UpdateReleaseBlocker{*channelBlocker}, plan.Blockers...)
	}
	if len(plan.Blockers) > 0 {
		if opts.AsJSON {
			if err := printUpdateReleasePlanJSON(cmd, "update_release_binary", plan, true); err != nil {
				return err
			}
		} else {
			printUpdateReleasePlan(cmd, "update_release_binary", plan, true)
		}
		return newExitCodeError(1, fmt.Errorf("gormes release update blocked"))
	}
	if !plan.UpdateAvailable {
		report := runUpdateReleaseWithServiceCoordination(cmd.Context(), opts, func(ctx context.Context) cli.UpdateReleaseBinaryReport {
			report := cli.UpdateReleaseBinaryReport{
				PreviousVersion: Version,
				NewVersion:      Version,
				Evidence: []cli.UpdateEvidence{{
					Kind:   cli.UpdateEvidenceCheckCurrent,
					Detail: fmt.Sprintf("already current at %s", Version),
				}},
			}
			return runUpdateReleaseAssetSkillSyncForBinaryReport(ctx, plan, report, opts)
		})
		if updateReleaseReportHasMutationEvidence(report) {
			appendUpdateReleaseLedgerEvidence(&report)
		}
		if opts.AsJSON {
			if err := printUpdateReleaseBinaryReportJSON(cmd, "update_release_binary", report); err != nil {
				return err
			}
		} else {
			printUpdateReleaseBinaryReport(cmd, "update_release_binary", report)
		}
		if report.Failed {
			return newExitCodeError(1, fmt.Errorf("gormes release update failed"))
		}
		return nil
	}
	managedBin, err := resolveManagedBinaryPath()
	if err != nil {
		return err
	}
	publishedBin, err := resolvePublishedBinaryPath()
	if err != nil {
		return err
	}
	runUpdate := opts.RunReleaseUpdate
	if runUpdate == nil {
		runUpdate = cli.RunUpdateReleaseBinaryUpdate
	}
	report := runUpdateReleaseWithServiceCoordination(cmd.Context(), opts, func(ctx context.Context) cli.UpdateReleaseBinaryReport {
		report := runUpdate(ctx, cli.UpdateReleaseBinaryOptions{
			Plan:             plan,
			ManagedBinPath:   managedBin,
			PublishedBinPath: publishedBin,
			Force:            opts.Force,
		})
		if !report.Failed {
			report = runUpdateReleaseAssetSkillSyncForBinaryReport(ctx, plan, report, opts)
		}
		return report
	})
	appendUpdateReleaseLedgerEvidence(&report)
	if opts.AsJSON {
		if err := printUpdateReleaseBinaryReportJSON(cmd, "update_release_binary", report); err != nil {
			return err
		}
	} else {
		printUpdateReleaseBinaryReport(cmd, "update_release_binary", report)
	}
	if report.Failed {
		return newExitCodeError(1, fmt.Errorf("gormes release update failed"))
	}
	return nil
}

const (
	defaultUpdateReleaseServiceDrainTimeout  = 75 * time.Second
	defaultUpdateReleaseServiceHealthTimeout = 30 * time.Second
)

func runUpdateReleaseWithServiceCoordination(ctx context.Context, opts updateReleaseBinaryModeOptions, mutation func(context.Context) cli.UpdateReleaseBinaryReport) cli.UpdateReleaseBinaryReport {
	if opts.RunServiceCoordination == nil {
		return mutation(ctx)
	}
	var lock cli.UpdateLock
	if opts.ReleaseUpdateLock != nil {
		lock = opts.ReleaseUpdateLock()
	}
	var services []cli.UpdateManagedService
	if opts.ReleaseManagedServices != nil {
		services = opts.ReleaseManagedServices()
	}
	var unmanaged []cli.UpdateUnmanagedSession
	if opts.ReleaseUnmanagedSessions != nil {
		unmanaged = opts.ReleaseUnmanagedSessions(ctx)
	}
	return opts.RunServiceCoordination(ctx, cli.UpdateServiceCoordinationOptions{
		Lock:              lock,
		Services:          services,
		UnmanagedSessions: unmanaged,
		Force:             opts.Force,
		DrainTimeout:      defaultUpdateReleaseServiceDrainTimeout,
		HealthTimeout:     defaultUpdateReleaseServiceHealthTimeout,
		Mutation:          mutation,
	})
}

func runUpdateReleaseAssetSkillSyncForBinaryReport(ctx context.Context, plan cli.UpdateReleasePlan, report cli.UpdateReleaseBinaryReport, opts updateReleaseBinaryModeOptions) cli.UpdateReleaseBinaryReport {
	if opts.LoadAssetManifest == nil || opts.RunAssetSkillSync == nil {
		return report
	}
	manifest, payloadRoot, err := opts.LoadAssetManifest(ctx, plan)
	if err != nil {
		report.Failed = true
		report.Evidence = append(report.Evidence, cli.UpdateEvidence{Kind: cli.UpdateEvidenceReleaseManifestFailed, Detail: err.Error()})
		return report
	}
	if len(manifest.Assets) == 0 && len(manifest.Skills) == 0 {
		return report
	}
	assetRoot := ""
	if opts.ReleaseAssetRoot != nil {
		var err error
		assetRoot, err = opts.ReleaseAssetRoot()
		if err != nil {
			report.Failed = true
			report.Evidence = append(report.Evidence, cli.UpdateEvidence{Kind: cli.UpdateEvidenceReleaseAssetSyncFailed, Detail: err.Error()})
			return report
		}
	}
	var profiles []skills.SkillProfileRoot
	if opts.ReleaseSkillProfiles != nil {
		var err error
		profiles, err = opts.ReleaseSkillProfiles()
		if err != nil {
			report.Failed = true
			report.Evidence = append(report.Evidence, cli.UpdateEvidence{Kind: cli.UpdateEvidenceReleaseSkillSyncFailed, Detail: err.Error()})
			return report
		}
	}
	snapshotPath := plan.SnapshotPath
	if snapshotPath != "" {
		snapshotPath = filepath.Join(snapshotPath, "assets-skills")
	}
	syncReport := opts.RunAssetSkillSync(ctx, cli.UpdateReleaseAssetSkillSyncOptions{
		Plan:          plan,
		Manifest:      manifest,
		PayloadRoot:   payloadRoot,
		AssetRoot:     assetRoot,
		SnapshotPath:  snapshotPath,
		SkillProfiles: profiles,
	})
	report.Evidence = append(report.Evidence, syncReport.Evidence...)
	if syncReport.Failed {
		report.Failed = true
		if syncReport.SnapshotPath != "" {
			report.OperatorRecovery = "asset/skill rollback may be needed from " + syncReport.SnapshotPath
		}
	}
	return report
}

func updateReleaseReportHasMutationEvidence(report cli.UpdateReleaseBinaryReport) bool {
	for _, ev := range report.Evidence {
		switch ev.Kind {
		case cli.UpdateEvidenceReleaseSwapCompleted,
			cli.UpdateEvidenceReleaseAssetSyncCompleted,
			cli.UpdateEvidenceReleaseSkillSyncCompleted:
			return true
		}
	}
	return false
}

func appendUpdateReleaseLedgerEvidence(report *cli.UpdateReleaseBinaryReport) {
	path, err := appendUpdateReleaseLedger(*report)
	if err != nil {
		report.Evidence = append(report.Evidence, cli.UpdateEvidence{Kind: cli.UpdateEvidenceLedgerUnavailable, Detail: err.Error()})
		return
	}
	report.Evidence = append(report.Evidence, cli.UpdateEvidence{Kind: cli.UpdateEvidenceLedgerAppended, Detail: path})
}

func normalizeUpdateReleaseChannel(channel cli.UpdateReleaseChannel) (cli.UpdateReleaseChannel, *cli.UpdateReleaseBlocker) {
	raw := strings.ToLower(strings.TrimSpace(string(channel)))
	switch raw {
	case "", string(cli.UpdateReleaseChannelStable):
		return cli.UpdateReleaseChannelStable, nil
	case "dev", string(cli.UpdateReleaseChannelDevelopment):
		return cli.UpdateReleaseChannelDevelopment, nil
	default:
		return cli.UpdateReleaseChannel(raw), &cli.UpdateReleaseBlocker{
			Kind:   cli.UpdateReleaseBlockerUnsupportedInstallState,
			Detail: fmt.Sprintf("unsupported update channel %q; use stable or development", channel),
		}
	}
}

type updateReleasePlanReportJSON struct {
	Build  buildProvenanceJSON   `json:"build"`
	Action string                `json:"action"`
	Failed bool                  `json:"failed"`
	Plan   cli.UpdateReleasePlan `json:"plan"`
}

type updateReleaseBinaryReportJSON struct {
	Build            buildProvenanceJSON  `json:"build"`
	Action           string               `json:"action"`
	Failed           bool                 `json:"failed"`
	SnapshotID       string               `json:"snapshot_id,omitempty"`
	SnapshotPath     string               `json:"snapshot_path,omitempty"`
	PreviousVersion  string               `json:"previous_version,omitempty"`
	NewVersion       string               `json:"new_version,omitempty"`
	ManagedBinPath   string               `json:"managed_bin_path,omitempty"`
	PublishedBinPath string               `json:"published_bin_path,omitempty"`
	Evidence         []updateEvidenceJSON `json:"evidence,omitempty"`
	OperatorRecovery string               `json:"operator_recovery,omitempty"`
}

func printUpdateReleasePlanJSON(cmd *cobra.Command, action string, plan cli.UpdateReleasePlan, failed bool) error {
	body, err := json.MarshalIndent(updateReleasePlanReportJSON{
		Build:  newBuildProvenance(),
		Action: action,
		Failed: failed,
		Plan:   plan,
	}, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(body))
	return nil
}

func printUpdateReleaseBinaryReportJSON(cmd *cobra.Command, action string, report cli.UpdateReleaseBinaryReport) error {
	evidence := make([]updateEvidenceJSON, 0, len(report.Evidence))
	for _, ev := range report.Evidence {
		evidence = append(evidence, updateEvidenceJSON{Kind: string(ev.Kind), Detail: ev.Detail})
	}
	body, err := json.MarshalIndent(updateReleaseBinaryReportJSON{
		Build:            newBuildProvenance(),
		Action:           action,
		Failed:           report.Failed,
		SnapshotID:       report.SnapshotID,
		SnapshotPath:     report.SnapshotPath,
		PreviousVersion:  report.PreviousVersion,
		NewVersion:       report.NewVersion,
		ManagedBinPath:   report.ManagedBinPath,
		PublishedBinPath: report.PublishedBinPath,
		Evidence:         evidence,
		OperatorRecovery: report.OperatorRecovery,
	}, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(body))
	return nil
}

func printUpdateReleasePlan(cmd *cobra.Command, action string, plan cli.UpdateReleasePlan, failed bool) {
	out := cmd.OutOrStdout()
	bannerText := "⚕ Planning Gormes Agent update..."
	successText := "✓ Update dry-run complete"
	if action == "update_check" {
		bannerText = "⚕ Checking Gormes Agent..."
		successText = "✓ Update check complete"
	}
	fmt.Fprintln(out, cli.Bold(out, bannerText))
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s %s\n", cli.Dim(out, "install kind:"), plan.InstallKind)
	fmt.Fprintf(out, "%s %s\n", cli.Dim(out, "source:"), plan.Source)
	fmt.Fprintf(out, "%s %s\n", cli.Dim(out, "channel:"), plan.Channel)
	fmt.Fprintf(out, "%s %s (%s)\n", cli.Dim(out, "current:"), plan.Current.Version, plan.Current.GitCommit)
	if plan.Target.Version != "" || plan.Target.GitCommit != "" {
		fmt.Fprintf(out, "%s %s (%s)\n", cli.Dim(out, "target:"), plan.Target.Version, plan.Target.GitCommit)
	}
	if plan.ArtifactName != "" {
		fmt.Fprintf(out, "%s %s\n", cli.Dim(out, "artifact:"), plan.ArtifactName)
	}
	if plan.SnapshotPath != "" {
		fmt.Fprintf(out, "%s %s\n", cli.Dim(out, "snapshot:"), plan.SnapshotPath)
	}
	if len(plan.Components) > 0 {
		parts := make([]string, 0, len(plan.Components))
		for _, component := range plan.Components {
			parts = append(parts, string(component))
		}
		fmt.Fprintf(out, "%s %s\n", cli.Dim(out, "components:"), strings.Join(parts, ", "))
	}
	if plan.UpdateAvailable {
		fmt.Fprintf(out, "%s yes\n", cli.Dim(out, "update available:"))
	} else {
		fmt.Fprintf(out, "%s no\n", cli.Dim(out, "update available:"))
	}
	for _, blocker := range plan.Blockers {
		detail := strings.TrimSpace(blocker.Detail)
		if detail == "" {
			fmt.Fprintf(out, "◆ %s\n", blocker.Kind)
			continue
		}
		fmt.Fprintf(out, "◆ %s\t%s\n", blocker.Kind, detail)
	}
	fmt.Fprintln(out)
	if failed {
		fmt.Fprintln(out, cli.Bold(out, "✗ Update planner blocked"))
		return
	}
	fmt.Fprintln(out, cli.Green(out, cli.Bold(out, successText)))
}

func printUpdateReleaseBinaryReport(cmd *cobra.Command, action string, report cli.UpdateReleaseBinaryReport) {
	out := cmd.OutOrStdout()
	bannerText := "⚕ Updating Gormes release binary..."
	successText := "✓ Release binary update complete"
	if action == "update_rollback" {
		bannerText = "⚕ Rolling back Gormes release binary..."
		successText = "✓ Release binary rollback complete"
	}
	fmt.Fprintln(out, cli.Bold(out, bannerText))
	fmt.Fprintln(out)
	if report.PreviousVersion != "" || report.NewVersion != "" {
		fmt.Fprintf(out, "%s %s -> %s\n", cli.Dim(out, "version:"), report.PreviousVersion, report.NewVersion)
	}
	if report.SnapshotID != "" {
		fmt.Fprintf(out, "%s %s\n", cli.Dim(out, "snapshot:"), report.SnapshotID)
	}
	if report.SnapshotPath != "" {
		fmt.Fprintf(out, "%s %s\n", cli.Dim(out, "snapshot path:"), report.SnapshotPath)
	}
	if report.ManagedBinPath != "" {
		fmt.Fprintf(out, "%s %s\n", cli.Dim(out, "managed binary:"), report.ManagedBinPath)
	}
	if report.PublishedBinPath != "" {
		fmt.Fprintf(out, "%s %s\n", cli.Dim(out, "published binary:"), report.PublishedBinPath)
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
		fmt.Fprintln(out, cli.Bold(out, "✗ Release binary update failed"))
	} else {
		fmt.Fprintln(out, cli.Green(out, cli.Bold(out, successText)))
	}
	if report.OperatorRecovery != "" {
		fmt.Fprintln(cmd.ErrOrStderr(), report.OperatorRecovery)
	}
}

func defaultLoadReleaseMetadata(ctx context.Context, channel cli.UpdateReleaseChannel) (cli.UpdateReleaseMetadata, error) {
	if channel != cli.UpdateReleaseChannelStable {
		return cli.UpdateReleaseMetadata{}, nil
	}
	apiURL := strings.TrimSpace(os.Getenv("GORMES_RELEASES_API_URL"))
	if apiURL == "" {
		apiURL = "https://api.github.com/repos/TrebuchetDynamics/gormes-agent/releases/latest"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return cli.UpdateReleaseMetadata{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return cli.UpdateReleaseMetadata{}, fmt.Errorf("fetch latest release metadata: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return cli.UpdateReleaseMetadata{}, fmt.Errorf("fetch latest release metadata: HTTP %s", resp.Status)
	}
	var body struct {
		TagName         string `json:"tag_name"`
		TargetCommitish string `json:"target_commitish"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil {
		return cli.UpdateReleaseMetadata{}, fmt.Errorf("decode latest release metadata: %w", err)
	}
	version := strings.TrimPrefix(strings.TrimSpace(body.TagName), "v")
	if version == "" {
		return cli.UpdateReleaseMetadata{}, fmt.Errorf("latest release metadata missing tag_name")
	}
	return cli.UpdateReleaseMetadata{
		Version:   version,
		Tag:       strings.TrimSpace(body.TagName),
		GitCommit: strings.TrimSpace(body.TargetCommitish),
	}, nil
}

func defaultLoadReleaseAssetManifest(ctx context.Context, plan cli.UpdateReleasePlan) (cli.UpdateReleaseManifest, string, error) {
	payloadRoot := strings.TrimSpace(os.Getenv("GORMES_RELEASE_PAYLOAD_ROOT"))
	if payloadRoot == "" {
		home, err := resolveManagedInstallHome()
		if err != nil {
			return cli.UpdateReleaseManifest{}, "", err
		}
		version := strings.TrimPrefix(strings.TrimSpace(plan.Target.Version), "v")
		if version == "" {
			version = "current"
		}
		payloadRoot = filepath.Join(home, "release-payloads", version)
	}
	manifestPath := strings.TrimSpace(os.Getenv("GORMES_RELEASE_MANIFEST_PATH"))
	if manifestPath == "" {
		manifestPath = filepath.Join(payloadRoot, "gormes-release-manifest.json")
	}
	file, err := os.Open(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cli.UpdateReleaseManifest{SchemaVersion: 1}, payloadRoot, nil
		}
		return cli.UpdateReleaseManifest{}, payloadRoot, err
	}
	defer file.Close()
	var manifest cli.UpdateReleaseManifest
	if err := json.NewDecoder(io.LimitReader(file, 8<<20)).Decode(&manifest); err != nil {
		return cli.UpdateReleaseManifest{}, payloadRoot, fmt.Errorf("decode release manifest: %w", err)
	}
	return manifest, payloadRoot, nil
}

func resolveReleaseAssetRoot() (string, error) {
	home, err := resolveManagedInstallHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "assets"), nil
}

func resolvePlannedUpdateSnapshotPath() (string, error) {
	root, err := resolveUpdateReleaseSnapshotRoot()
	if err != nil {
		return "", err
	}
	name := time.Now().UTC().Format("20060102-150405") + "-pre-update"
	return filepath.Join(root, name), nil
}

func resolveUpdateReleaseSnapshotRoot() (string, error) {
	home, err := resolveManagedInstallHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "snapshots"), nil
}

func defaultReleaseUpdateLock() cli.UpdateLock {
	home, err := resolveManagedInstallHome()
	if err != nil {
		return cli.NewFileUpdateLock(filepath.Join(os.TempDir(), "gormes-update.lock"), fmt.Sprintf("pid=%d", os.Getpid()))
	}
	return cli.NewFileUpdateLock(filepath.Join(home, "update.lock"), fmt.Sprintf("pid=%d", os.Getpid()))
}

const profileFleetUpdateManagedServiceName = "gormes-profile-fleet"

func defaultReleaseManagedServices() []cli.UpdateManagedService {
	if services := defaultReleaseProfileFleetManagedServices(); len(services) > 0 {
		return services
	}
	if runtime.GOOS != "linux" {
		return nil
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		return nil
	}
	if _, err := os.Stat(defaultReleaseGatewayServiceUnitPath()); err != nil {
		return nil
	}
	return []cli.UpdateManagedService{systemdUpdateManagedService{
		name:    defaultGatewayServiceName,
		manager: systemdUpdateServiceManager{},
	}}
}

func defaultReleaseProfileFleetManagedServices() []cli.UpdateManagedService {
	cfg, err := config.Load(nil)
	if err != nil || !cfg.ProfileConfigV2Available() {
		return nil
	}
	supervisor := newUpdateFleetSupervisor(cfg)
	if supervisor == nil {
		return nil
	}
	return []cli.UpdateManagedService{profileFleetUpdateManagedService{supervisor: supervisor}}
}

type profileFleetUpdateManagedService struct {
	supervisor updateFleetSupervisor
}

func (s profileFleetUpdateManagedService) UpdateServiceName() string {
	return profileFleetUpdateManagedServiceName
}

func (s profileFleetUpdateManagedService) UpdateServiceRunning(ctx context.Context) (bool, error) {
	status, err := s.supervisor.Status(ctx)
	if err != nil {
		return false, fmt.Errorf("profile fleet status: %w", err)
	}
	return updateProfileFleetHasLiveGateway(status), nil
}

func (profileFleetUpdateManagedService) DrainUpdateService(context.Context, time.Duration) error {
	return nil
}

func (s profileFleetUpdateManagedService) StopUpdateService(ctx context.Context) error {
	report, err := s.supervisor.StopAll(ctx)
	if err != nil {
		return fmt.Errorf("profile fleet stop-all: %w", err)
	}
	return profileFleetOperationReportError(gateway.FleetOperationStopAll, report)
}

func (s profileFleetUpdateManagedService) StartUpdateService(ctx context.Context) error {
	report, err := s.supervisor.StartAll(ctx)
	if err != nil {
		return fmt.Errorf("profile fleet start-all: %w", err)
	}
	return profileFleetOperationReportError(gateway.FleetOperationStartAll, report)
}

func (s profileFleetUpdateManagedService) HealthCheckUpdateService(ctx context.Context, timeout time.Duration) error {
	started := time.Now()
	for {
		if err := profileFleetHealthCheck(ctx, s.supervisor); err == nil {
			return nil
		} else if timeout <= 0 || time.Since(started) >= timeout {
			return err
		}
		wait := gatewayRestartPollInterval
		if remaining := timeout - time.Since(started); remaining < wait {
			wait = remaining
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func profileFleetHealthCheck(ctx context.Context, supervisor updateFleetSupervisor) error {
	status, err := supervisor.Status(ctx)
	if err != nil {
		return fmt.Errorf("profile fleet health status: %w", err)
	}
	enabled := 0
	for _, profile := range status.Profiles {
		if !profile.Enabled {
			continue
		}
		enabled++
		if !profile.Runtime.Live {
			return fmt.Errorf("profile fleet health: profile %s is not live", profile.ProfileID)
		}
	}
	if enabled == 0 {
		return fmt.Errorf("profile fleet health: no enabled profiles")
	}
	return nil
}

func profileFleetOperationReportError(action gateway.FleetOperation, report gateway.FleetOperationReport) error {
	if report.Summary.Failed == 0 && report.Summary.Unavailable == 0 {
		return nil
	}
	return fmt.Errorf("profile fleet %s incomplete: targeted=%d succeeded=%d unavailable=%d failed=%d", action, report.Summary.TargetedProfiles, report.Summary.Succeeded, report.Summary.Unavailable, report.Summary.Failed)
}

func defaultReleaseUnmanagedSessions(ctx context.Context) []cli.UpdateUnmanagedSession {
	snapshot, err := readUpdateGatewayRuntimeSnapshot(ctx)
	if err != nil || !snapshot.Validation.Live || snapshot.Status.ActiveAgents <= 0 {
		return nil
	}
	if defaultReleaseGatewayServiceRunning(ctx) {
		return nil
	}
	return []cli.UpdateUnmanagedSession{{
		PID:     snapshot.Validation.PID,
		Command: snapshot.Validation.Command,
		Detail:  fmt.Sprintf("live gateway has active_agents=%d outside the managed service", snapshot.Status.ActiveAgents),
	}}
}

func defaultReleaseGatewayServiceRunning(ctx context.Context) bool {
	services := defaultReleaseManagedServices()
	if len(services) == 0 {
		return false
	}
	running, err := services[0].UpdateServiceRunning(ctx)
	return err == nil && running
}

func defaultReleaseGatewayServiceUnitPath() string {
	configHome := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if configHome == "" {
		if dir, err := os.UserConfigDir(); err == nil {
			configHome = dir
		}
	}
	if configHome == "" {
		return ""
	}
	return filepath.Join(configHome, "systemd", "user", defaultGatewayServiceName)
}

type systemdUpdateServiceManager struct{}

func (systemdUpdateServiceManager) Stop(ctx context.Context, service string) error {
	command := exec.CommandContext(ctx, "systemctl", "--user", "stop", service)
	out, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl --user stop %s: %w: %s", service, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (systemdUpdateServiceManager) Start(ctx context.Context, service string) error {
	command := exec.CommandContext(ctx, "systemctl", "--user", "start", service)
	out, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl --user start %s: %w: %s", service, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (systemdUpdateServiceManager) ServiceActiveStatus(service string) (cli.ServiceActiveStatusCheck, error) {
	command := exec.Command("systemctl", "--user", "is-active", service)
	out, err := command.CombinedOutput()
	raw := strings.TrimSpace(string(out))
	status := parseSystemdGatewayActiveStatus(raw)
	check := cli.ServiceActiveStatusCheck{Status: status, Raw: raw}
	if err != nil && status == cli.ServiceActiveStatusUnknown {
		check.Unavailable = true
		check.Detail = err.Error()
	}
	return check, nil
}

type systemdUpdateManagedService struct {
	name    string
	manager systemdUpdateServiceManager
}

func (s systemdUpdateManagedService) UpdateServiceName() string {
	if strings.TrimSpace(s.name) == "" {
		return defaultGatewayServiceName
	}
	return s.name
}

func (s systemdUpdateManagedService) UpdateServiceRunning(context.Context) (bool, error) {
	check, err := s.manager.ServiceActiveStatus(s.UpdateServiceName())
	if err != nil {
		return false, err
	}
	if check.Unavailable {
		detail := strings.TrimSpace(check.Detail)
		if detail == "" {
			detail = "service manager unavailable"
		}
		return false, fmt.Errorf("%s status unavailable: %s", s.UpdateServiceName(), detail)
	}
	switch check.Status {
	case cli.ServiceActiveStatusActive, cli.ServiceActiveStatusActivating:
		return true, nil
	case cli.ServiceActiveStatusInactive, cli.ServiceActiveStatusFailed, cli.ServiceActiveStatusUnknown:
		return false, nil
	default:
		return false, nil
	}
}

func (s systemdUpdateManagedService) DrainUpdateService(ctx context.Context, timeout time.Duration) error {
	if ctx == nil {
		ctx = context.Background()
	}
	started := time.Now()
	for {
		snapshot, err := readUpdateGatewayRuntimeSnapshot(ctx)
		if err != nil || !snapshot.Validation.Live || snapshot.Status.ActiveAgents == 0 {
			return nil
		}
		if timeout <= 0 || time.Since(started) >= timeout {
			return fmt.Errorf("%s still has active_agents=%d after drain timeout %s", s.UpdateServiceName(), snapshot.Status.ActiveAgents, timeout)
		}
		wait := 500 * time.Millisecond
		if remaining := timeout - time.Since(started); remaining < wait {
			wait = remaining
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (s systemdUpdateManagedService) StopUpdateService(ctx context.Context) error {
	return s.manager.Stop(ctx, s.UpdateServiceName())
}

func (s systemdUpdateManagedService) StartUpdateService(ctx context.Context) error {
	return s.manager.Start(ctx, s.UpdateServiceName())
}

func (s systemdUpdateManagedService) HealthCheckUpdateService(_ context.Context, timeout time.Duration) error {
	poll := cli.PollServiceRestartActive(cli.ServiceRestartPollOptions{
		Service:      s.UpdateServiceName(),
		Runner:       s.manager,
		BaseTimeout:  timeout,
		PollInterval: gatewayRestartPollInterval,
	})
	if poll.Outcome != cli.ServiceRestartPollRestarted {
		return fmt.Errorf("service did not become active after update restart (outcome=%s)", poll.Outcome)
	}
	return nil
}

func detectUpdateInstallKind() cli.UpdateInstallKind {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("GORMES_INSTALL_METHOD"))) {
	case "release", "binary-fetch", "github_release", "github-releases":
		return cli.UpdateInstallKindRelease
	case "source", "source-build", "managed_source", "managed-source":
		return cli.UpdateInstallKindManagedSource
	}
	managedBin, managedBinErr := resolveManagedBinaryPath()
	if managedBinErr == nil {
		if exe, err := os.Executable(); err == nil && sameUpdatePath(exe, managedBin) {
			return cli.UpdateInstallKindRelease
		}
	}
	publishedBin, publishedBinErr := resolvePublishedBinaryPath()
	if publishedBinErr == nil {
		if exe, err := os.Executable(); err == nil && sameUpdatePath(exe, publishedBin) {
			return cli.UpdateInstallKindRelease
		}
	}
	checkoutDir, checkoutErr := resolveManagedCheckoutDir()
	if checkoutErr == nil && isGitWorktreeDir(checkoutDir) {
		return cli.UpdateInstallKindManagedSource
	}
	if cwd, err := os.Getwd(); err == nil && isGormesSourceCheckout(cwd) {
		return cli.UpdateInstallKindUnmanagedSource
	}
	return cli.UpdateInstallKindUnknown
}

func isGormesSourceCheckout(dir string) bool {
	if !isGitWorktreeDir(dir) {
		return false
	}
	body, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return false
	}
	return strings.Contains(string(body), "module github.com/TrebuchetDynamics/gormes-agent")
}

func isGitWorktreeDir(dir string) bool {
	if strings.TrimSpace(dir) == "" {
		return false
	}
	if stat, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		return stat.IsDir() || stat.Mode().IsRegular()
	}
	return false
}

func sameUpdatePath(left, right string) bool {
	left = cleanUpdatePath(left)
	right = cleanUpdatePath(right)
	return left != "" && right != "" && left == right
}

func cleanUpdatePath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return filepath.Clean(path)
}

func attachUpdateLogMirror(cmd *cobra.Command, report *cli.UpdateReport) func() {
	path, err := resolveUpdateLogPath()
	if err != nil {
		report.Evidence = append(report.Evidence, cli.UpdateEvidence{Kind: cli.UpdateEvidenceHangupLogUnavailable, Detail: err.Error()})
		return func() {}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		report.Evidence = append(report.Evidence, cli.UpdateEvidence{Kind: cli.UpdateEvidenceHangupLogUnavailable, Detail: err.Error()})
		return func() {}
	}
	logFile, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		report.Evidence = append(report.Evidence, cli.UpdateEvidence{Kind: cli.UpdateEvidenceHangupLogUnavailable, Detail: err.Error()})
		return func() {}
	}
	report.Evidence = append(report.Evidence, cli.UpdateEvidence{Kind: cli.UpdateEvidenceHangupLogMirrored, Detail: path})
	cmd.SetOut(cli.NewUpdateOutputMirror(cmd.OutOrStdout(), logFile))
	return func() { _ = logFile.Close() }
}

func appendUpdateLedgerEvidence(report *cli.UpdateReport, restartGateway string) {
	path, err := appendUpdateLedger(*report, restartGateway)
	if err != nil {
		report.Evidence = append(report.Evidence, cli.UpdateEvidence{Kind: cli.UpdateEvidenceLedgerUnavailable, Detail: err.Error()})
		return
	}
	report.Evidence = append(report.Evidence, cli.UpdateEvidence{Kind: cli.UpdateEvidenceLedgerAppended, Detail: path})
}

type updateLedgerEvent struct {
	Event          string               `json:"event"`
	Timestamp      string               `json:"timestamp"`
	Build          buildProvenanceJSON  `json:"build"`
	Branch         string               `json:"branch"`
	PreviousBranch string               `json:"previous_branch,omitempty"`
	Failed         bool                 `json:"failed"`
	RestartGateway string               `json:"restart_gateway,omitempty"`
	Evidence       []updateEvidenceJSON `json:"evidence,omitempty"`
}

type updateReleaseLedgerEvent struct {
	Event            string               `json:"event"`
	Timestamp        string               `json:"timestamp"`
	Build            buildProvenanceJSON  `json:"build"`
	Failed           bool                 `json:"failed"`
	SnapshotID       string               `json:"snapshot_id,omitempty"`
	SnapshotPath     string               `json:"snapshot_path,omitempty"`
	PreviousVersion  string               `json:"previous_version,omitempty"`
	NewVersion       string               `json:"new_version,omitempty"`
	ManagedBinPath   string               `json:"managed_bin_path,omitempty"`
	PublishedBinPath string               `json:"published_bin_path,omitempty"`
	Evidence         []updateEvidenceJSON `json:"evidence,omitempty"`
	OperatorRecovery string               `json:"operator_recovery,omitempty"`
}

func appendUpdateLedger(report cli.UpdateReport, restartGateway string) (string, error) {
	path, err := resolveUpdateLedgerPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	event := updateLedgerEvent{
		Event:          "update",
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		Build:          newBuildProvenance(),
		Branch:         report.Branch,
		PreviousBranch: report.PreviousBranch,
		Failed:         report.Failed,
		RestartGateway: restartGateway,
		Evidence:       make([]updateEvidenceJSON, 0, len(report.Evidence)),
	}
	for _, evidence := range report.Evidence {
		event.Evidence = append(event.Evidence, updateEvidenceJSON{Kind: string(evidence.Kind), Detail: evidence.Detail})
	}
	body, err := json.Marshal(event)
	if err != nil {
		return "", err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return "", err
	}
	if _, err := file.Write(append(body, '\n')); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return path, nil
}

func appendUpdateReleaseLedger(report cli.UpdateReleaseBinaryReport) (string, error) {
	path, err := resolveUpdateLedgerPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	event := updateReleaseLedgerEvent{
		Event:            "release_update",
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
		Build:            newBuildProvenance(),
		Failed:           report.Failed,
		SnapshotID:       report.SnapshotID,
		SnapshotPath:     report.SnapshotPath,
		PreviousVersion:  report.PreviousVersion,
		NewVersion:       report.NewVersion,
		ManagedBinPath:   report.ManagedBinPath,
		PublishedBinPath: report.PublishedBinPath,
		Evidence:         make([]updateEvidenceJSON, 0, len(report.Evidence)),
		OperatorRecovery: report.OperatorRecovery,
	}
	for _, evidence := range report.Evidence {
		event.Evidence = append(event.Evidence, updateEvidenceJSON{Kind: string(evidence.Kind), Detail: evidence.Detail})
	}
	body, err := json.Marshal(event)
	if err != nil {
		return "", err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return "", err
	}
	if _, err := file.Write(append(body, '\n')); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return path, nil
}

func resolveUpdateLogPath() (string, error) {
	home, err := resolveManagedInstallHome()
	if err != nil {
		return "", fmt.Errorf("resolve update log: %w", err)
	}
	return filepath.Join(home, "lifecycle", "update.log"), nil
}

func resolveUpdateLedgerPath() (string, error) {
	home, err := resolveManagedInstallHome()
	if err != nil {
		return "", fmt.Errorf("resolve update ledger: %w", err)
	}
	return filepath.Join(home, "lifecycle", "install.log.jsonl"), nil
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
	home, err := resolveManagedInstallHome()
	if err != nil {
		return "", fmt.Errorf("resolve managed checkout: %w", err)
	}
	return filepath.Join(home, "gormes-agent"), nil
}

func defaultBinaryPublisherFor(checkoutDir string) cli.UpdateBinaryPublisher {
	return func(ctx context.Context, req cli.UpdateBinaryPublishRequest) cli.UpdateReport {
		opts, err := defaultUpdateBinaryPublishOptions(checkoutDir)
		if strings.TrimSpace(req.CheckoutDir) != "" {
			opts.CheckoutDir = req.CheckoutDir
		}
		if err != nil {
			return cli.UpdateReport{
				Failed: true,
				Evidence: []cli.UpdateEvidence{
					{Kind: cli.UpdateEvidencePublishFailed, Detail: err.Error()},
				},
			}
		}
		return cli.RunUpdateBinaryPublish(ctx, opts)
	}
}

func defaultUpdateGatewayRestartFor() cli.UpdateGatewayRestartRunner {
	return func(ctx context.Context, req cli.UpdateGatewayRestartRequest) cli.UpdateReport {
		return runUpdateGatewayRestartForPolicy(ctx, req.Policy)
	}
}

type updateFleetSupervisor interface {
	Status(context.Context) (gateway.FleetStatus, error)
	StartAll(context.Context) (gateway.FleetOperationReport, error)
	StopAll(context.Context) (gateway.FleetOperationReport, error)
	RestartAll(context.Context) (gateway.FleetOperationReport, error)
}

var newUpdateFleetSupervisor = func(cfg config.Config) updateFleetSupervisor {
	return gateway.NewFleetSupervisor(cfg, gateway.FleetSupervisorOptions{
		HomeRoot: config.GormesHome(),
		Worker:   gateway.NewCommandFleetWorker(gateway.CommandFleetWorkerOptions{}),
	})
}

func runUpdateGatewayRestartForPolicy(ctx context.Context, policy string) cli.UpdateReport {
	policy = strings.TrimSpace(policy)
	if policy == "" {
		policy = "auto"
	}
	switch policy {
	case "never", "auto", "always":
		if report, ok := updateGatewayRestartProfileFleet(ctx, policy); ok {
			return report
		}
	case "":
		policy = "auto"
	default:
		return cli.UpdateReport{
			Failed: true,
			Evidence: []cli.UpdateEvidence{
				{Kind: cli.UpdateEvidenceGatewayRestartUnavailable, Detail: fmt.Sprintf("invalid restart policy %q", policy)},
			},
		}
	}
	switch policy {
	case "never":
		return updateGatewayRestartNever(ctx)
	case "auto":
		return updateGatewayRestartAuto(ctx)
	case "always":
		return updateGatewayRestartRecorded(ctx, true)
	default:
		return cli.UpdateReport{}
	}
}

func updateGatewayRestartProfileFleet(ctx context.Context, policy string) (cli.UpdateReport, bool) {
	cfg, err := config.Load(nil)
	if err != nil || !cfg.ProfileConfigV2Available() {
		return cli.UpdateReport{}, false
	}
	supervisor := newUpdateFleetSupervisor(cfg)
	if supervisor == nil {
		return cli.UpdateReport{}, false
	}
	status, err := supervisor.Status(ctx)
	if err != nil {
		return updateProfileFleetRestartUnavailable(policy, fmt.Sprintf("profile fleet status unavailable: %v", err)), true
	}
	running := updateProfileFleetHasLiveGateway(status)
	if policy == "never" {
		if !running {
			return cli.UpdateReport{}, true
		}
		return cli.UpdateReport{Evidence: []cli.UpdateEvidence{{Kind: cli.UpdateEvidenceGatewayRestartNeeded, Detail: "profile fleet restart skipped by policy=never; restart all profile gateways manually to load the updated binary"}}}, true
	}
	if policy == "auto" && !running {
		return cli.UpdateReport{}, true
	}
	restart, err := supervisor.RestartAll(ctx)
	if err != nil {
		return updateProfileFleetRestartUnavailable(policy, "profile fleet restart-all failed"), true
	}
	return updateProfileFleetRestartReport(policy, restart), true
}

func updateProfileFleetHasLiveGateway(status gateway.FleetStatus) bool {
	for _, profile := range status.Profiles {
		if profile.Enabled && profile.Runtime.Live {
			return true
		}
	}
	return false
}

func updateProfileFleetRestartReport(policy string, report gateway.FleetOperationReport) cli.UpdateReport {
	detail := fmt.Sprintf("profile fleet restart-all targeted=%d succeeded=%d unavailable=%d failed=%d", report.Summary.TargetedProfiles, report.Summary.Succeeded, report.Summary.Unavailable, report.Summary.Failed)
	if report.Summary.TargetedProfiles > 0 && report.Summary.Failed == 0 && report.Summary.Unavailable == 0 {
		return cli.UpdateReport{Evidence: []cli.UpdateEvidence{{Kind: cli.UpdateEvidenceGatewayRestarted, Detail: detail}}}
	}
	return updateProfileFleetRestartUnavailable(policy, detail)
}

func updateProfileFleetRestartUnavailable(policy, detail string) cli.UpdateReport {
	report := cli.UpdateReport{Evidence: []cli.UpdateEvidence{{Kind: cli.UpdateEvidenceGatewayRestartUnavailable, Detail: detail}}}
	if policy == "always" {
		report.Failed = true
	}
	return report
}

func updateGatewayRestartNever(ctx context.Context) cli.UpdateReport {
	snapshot, err := readUpdateGatewayRuntimeSnapshot(ctx)
	if err != nil || !snapshot.Validation.Live {
		return cli.UpdateReport{}
	}
	return cli.UpdateReport{Evidence: []cli.UpdateEvidence{{
		Kind:   cli.UpdateEvidenceGatewayRestartNeeded,
		Detail: "live gateway was not restarted because --restart-gateway=never",
	}}}
}

func updateGatewayRestartAuto(ctx context.Context) cli.UpdateReport {
	snapshot, err := readUpdateGatewayRuntimeSnapshot(ctx)
	if err != nil {
		return cli.UpdateReport{Evidence: []cli.UpdateEvidence{{
			Kind:   cli.UpdateEvidenceGatewayRestartUnavailable,
			Detail: fmt.Sprintf("gateway runtime validation unavailable: %v", err),
		}}}
	}
	if !snapshot.Validation.Live {
		return cli.UpdateReport{}
	}
	if snapshot.Status.ActiveAgents > 0 {
		return cli.UpdateReport{Evidence: []cli.UpdateEvidence{{
			Kind:   cli.UpdateEvidenceGatewayRestartUnavailable,
			Detail: fmt.Sprintf("live gateway has active_agents=%d; restart skipped by policy=auto", snapshot.Status.ActiveAgents),
		}}}
	}
	return updateGatewayRestartRecorded(ctx, false)
}

func updateGatewayRestartRecorded(ctx context.Context, failOnError bool) cli.UpdateReport {
	report, err := restartRecordedGatewayRuntime(ctx, defaultGatewayStopTimeout)
	if err == nil {
		detail := "gateway restarted"
		if report.OldPID > 0 && report.NewPID > 0 {
			detail = fmt.Sprintf("gateway restarted pid=%d -> %d", report.OldPID, report.NewPID)
		} else if report.NewPID > 0 {
			detail = fmt.Sprintf("gateway started pid=%d", report.NewPID)
		}
		return cli.UpdateReport{Evidence: []cli.UpdateEvidence{{Kind: cli.UpdateEvidenceGatewayRestarted, Detail: detail}}}
	}
	kind := cli.UpdateEvidenceGatewayRestartUnavailable
	if strings.Contains(err.Error(), "timed out") {
		kind = cli.UpdateEvidenceGatewayRestartTimeout
	}
	return cli.UpdateReport{
		Failed: failOnError,
		Evidence: []cli.UpdateEvidence{{
			Kind:   kind,
			Detail: err.Error(),
		}},
	}
}

func readUpdateGatewayRuntimeSnapshot(ctx context.Context) (gateway.RuntimeStatusSnapshot, error) {
	store := newGatewayRestartRuntimeStore(config.GatewayRuntimeStatusPath())
	return store.ReadValidatedRuntimeStatusSnapshot(ctx)
}

func defaultUpdateBinaryPublishOptions(checkoutDir string) (cli.UpdateBinaryPublishOptions, error) {
	managedBin, err := resolveManagedBinaryPath()
	if err != nil {
		return cli.UpdateBinaryPublishOptions{}, err
	}
	publishedBin, err := resolvePublishedBinaryPath()
	if err != nil {
		return cli.UpdateBinaryPublishOptions{}, err
	}
	return cli.UpdateBinaryPublishOptions{
		CheckoutDir:       checkoutDir,
		ManagedBinPath:    managedBin,
		PublishedBinPath:  publishedBin,
		ActivePathPath:    resolveActiveGormesCommandPath(),
		RefreshActivePath: !updateSandboxBinDirSet(),
		Runner:            cli.RealUpdateCommandRunner{},
		Git:               cli.RealUpdateGitRunner{},
	}, nil
}

func resolveManagedInstallHome() (string, error) {
	if home := strings.TrimSpace(os.Getenv("GORMES_INSTALL_HOME")); home != "" {
		return home, nil
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userHome, ".gormes"), nil
}

func resolveManagedBinaryPath() (string, error) {
	home, err := resolveManagedInstallHome()
	if err != nil {
		return "", fmt.Errorf("resolve managed binary: %w", err)
	}
	return filepath.Join(home, "bin", gormesExecutableName()), nil
}

func resolvePublishedBinaryPath() (string, error) {
	if dir := strings.TrimSpace(os.Getenv("GORMES_BIN_DIR")); dir != "" {
		return filepath.Join(dir, gormesExecutableName()), nil
	}
	if prefix := strings.TrimSpace(os.Getenv("GORMES_PREFIX")); prefix != "" {
		return filepath.Join(prefix, "bin", gormesExecutableName()), nil
	}
	if runtime.GOOS == "android" {
		if prefix := strings.TrimSpace(os.Getenv("PREFIX")); prefix != "" {
			return filepath.Join(prefix, "bin", gormesExecutableName()), nil
		}
	}
	if runtime.GOOS == "linux" && os.Geteuid() == 0 {
		return filepath.Join("/usr", "local", "bin", gormesExecutableName()), nil
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve published binary: %w", err)
	}
	return filepath.Join(userHome, ".local", "bin", gormesExecutableName()), nil
}

func resolveActiveGormesCommandPath() string {
	active, err := exec.LookPath("gormes")
	if err != nil {
		return ""
	}
	if strings.TrimSpace(active) == "" {
		return ""
	}
	if filepath.IsAbs(active) {
		return active
	}
	abs, err := filepath.Abs(active)
	if err != nil {
		return active
	}
	return abs
}

func updateSandboxBinDirSet() bool {
	return strings.TrimSpace(os.Getenv("GORMES_BIN_DIR")) != "" || strings.TrimSpace(os.Getenv("GORMES_PREFIX")) != ""
}

func gormesExecutableName() string {
	if runtime.GOOS == "windows" {
		return "gormes.exe"
	}
	return "gormes"
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
				{Name: config.DefaultProfileID, Root: profileRoot},
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
		dest := filepath.Join(home, "lifecycle", "backups", "pre-update-"+stamp+".zip")
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
