package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type UpdateEvidenceKind string

const (
	UpdateEvidenceCheck                     UpdateEvidenceKind = "update_check"
	UpdateEvidenceNotManagedCheckout        UpdateEvidenceKind = "update_not_managed_checkout"
	UpdateEvidenceAutostashCreated          UpdateEvidenceKind = "update_autostash_created"
	UpdateEvidenceAutostashRestored         UpdateEvidenceKind = "update_autostash_restored"
	UpdateEvidenceAutostashPreserved        UpdateEvidenceKind = "update_autostash_preserved"
	UpdateEvidenceBranchSwitched            UpdateEvidenceKind = "update_branch_switched"
	UpdateEvidenceDetachedHeadSwitched      UpdateEvidenceKind = "update_detached_head_switched"
	UpdateEvidenceBranchSwitchFailed        UpdateEvidenceKind = "update_branch_switch_failed"
	UpdateEvidenceNetworkError              UpdateEvidenceKind = "update_network_error"
	UpdateEvidenceAuthError                 UpdateEvidenceKind = "update_auth_error"
	UpdateEvidenceGitError                  UpdateEvidenceKind = "update_git_error"
	UpdateEvidenceResetFallback             UpdateEvidenceKind = "update_reset_fallback"
	UpdateEvidenceGatewayRestarted          UpdateEvidenceKind = "update_gateway_restarted"
	UpdateEvidenceGatewayRestartUnavailable UpdateEvidenceKind = "update_gateway_restart_unavailable"
	UpdateEvidenceGatewayRestartTimeout     UpdateEvidenceKind = "update_gateway_restart_timeout"
	UpdateEvidenceHangupIgnored             UpdateEvidenceKind = "update_hangup_ignored"
	UpdateEvidenceHangupUnavailable         UpdateEvidenceKind = "update_hangup_unavailable"
	UpdateEvidenceHangupLogMirrored         UpdateEvidenceKind = "update_hangup_log_mirrored"
	UpdateEvidenceHangupLogUnavailable      UpdateEvidenceKind = "update_hangup_log_unavailable"
	UpdateEvidenceStaleDashboardDetected    UpdateEvidenceKind = "update_stale_dashboard_detected"
	UpdateEvidenceStaleDashboardKilled      UpdateEvidenceKind = "update_stale_dashboard_killed"
	UpdateEvidenceStaleDashboardKillFailed  UpdateEvidenceKind = "update_stale_dashboard_kill_failed"
	UpdateEvidencePreBackupSkipped          UpdateEvidenceKind = "update_pre_backup_skipped"
	UpdateEvidencePreBackupRequested        UpdateEvidenceKind = "update_pre_backup_requested"
	UpdateEvidencePreBackupCompleted        UpdateEvidenceKind = "update_pre_backup_completed"
	UpdateEvidencePreBackupFailed           UpdateEvidenceKind = "update_pre_backup_failed"
	UpdateEvidenceSkillSyncCompleted        UpdateEvidenceKind = "update_skill_sync_completed"
	UpdateEvidenceSkillSyncFailed           UpdateEvidenceKind = "update_skill_sync_failed"
	UpdateEvidenceWebBuildCompleted         UpdateEvidenceKind = "update_web_build_completed"
	UpdateEvidenceWebBuildSkipped           UpdateEvidenceKind = "update_web_build_skipped"
	UpdateEvidenceWebBuildUnavailable       UpdateEvidenceKind = "update_web_build_unavailable"
	UpdateEvidenceWebBuildFailed            UpdateEvidenceKind = "update_web_build_failed"
	UpdateEvidenceConfigMigrateNeeded       UpdateEvidenceKind = "update_config_migrate_needed"
	UpdateEvidenceConfigMigrateCompleted    UpdateEvidenceKind = "update_config_migrate_completed"
	UpdateEvidenceConfigMigrateFailed       UpdateEvidenceKind = "update_config_migrate_failed"
	UpdateEvidenceRollbackHint              UpdateEvidenceKind = "update_rollback_hint"
)

// ConfigVersionResult is the abstract per-update return from a
// ConfigCheckRunner. It is decoupled from internal/config so the lifecycle
// package stays import-free; callers in cmd/gormes/update.go adapt
// internal/config.CheckReport into this shape.
type ConfigVersionResult struct {
	Current int
	Latest  int
}

// BackupResult is the outcome of a successful pre-update backup write.
// Path is operator-readable (e.g., ~/.gormes/backups/pre-update-<ts>.zip).
// SizeBytes is the byte count of the resulting archive. DurationMs is the
// total wall time the writer took (used for the "✓ Pre-update backup
// (4.0MB, 1.84s)" rendering in the structured progress UX).
// PrunedCount and PrunedBytes report how many older backups the writer
// also removed in the same call (zero when no retention pruning ran).
// FileCount is the number of regular files archived — surfaced in
// evidence so operators can spot a suspiciously small backup
// (e.g., a 4MB zip containing 3 files vs 200) before relying on it for
// rollback.
type BackupResult struct {
	Path        string
	SizeBytes   int64
	DurationMs  int64
	FileCount   int
	PrunedCount int
	PrunedBytes int64
}

// BackupWriter is the seam invoked by RunUpdateLifecycle when policy
// resolves to Requested AND the seam is non-nil. Returning a non-nil
// error emits update_pre_backup_failed but never fails the overall
// update (Hermes' "never raises" contract for backup). A nil seam in
// UpdateLifecycleOptions falls back to the policy-only behavior:
// emitting update_pre_backup_requested with a "writer not yet
// implemented" detail so existing transcripts and tests don't regress.
type BackupWriter func(ctx context.Context) (BackupResult, error)

// ConfigCheckRunner is the seam invoked by RunUpdateLifecycle after the
// web build step. It reports the on-disk config version vs. what this
// binary writes. Returning a non-nil error emits update_config_migrate_failed
// but never fails the overall update.
type ConfigCheckRunner func(ctx context.Context) (ConfigVersionResult, error)

// ConfigMigrateRunner is the seam invoked by RunUpdateLifecycle when the
// operator passed --yes (or equivalent) AND the check reports an outdated
// version. Returning a non-nil error emits update_config_migrate_failed
// but never fails the overall update.
type ConfigMigrateRunner func(ctx context.Context) error

// WebBuildResult classifies the outcome of an optional web UI build step.
// Exactly one of Skipped, Unavailable, or "completed" (neither flag set,
// no error from the runner) describes the run.
//
// Use Skipped for policy choices (--skip-web, no package.json).
// Use Unavailable for environment limitations (npm not on PATH).
// Use Detail for the success line ("web UI built in 1.2s").
// Use the error return for non-zero exit from npm install / npm run build.
type WebBuildResult struct {
	Skipped     bool
	Unavailable bool
	Reason      string
	Detail      string
}

// WebBuildRunner is the seam invoked by RunUpdateLifecycle after skill
// sync. Returning a non-nil error marks the build as failed but never
// fails the overall update (Hermes' soft-failure contract — only `hermes
// web` treats a build failure as fatal). A nil seam in
// UpdateLifecycleOptions disables the build entirely (silent default).
type WebBuildRunner func(ctx context.Context) (WebBuildResult, error)

// SkillSyncResult is the abstract per-update report returned by a
// SkillSyncRunner. It is decoupled from internal/skills so the lifecycle
// package stays import-free; callers in cmd/gormes/update.go adapt the
// internal/skills.BundledSkillProfileSyncReport into this shape.
type SkillSyncResult struct {
	Profiles []SkillSyncProfileResult
}

// SkillSyncProfileResult mirrors internal/skills.SkillProfileSyncSummary
// without importing it. Counts:
//
//	Added     — number of bundled skills newly written into this profile
//	Unchanged — bundled skills already present and identical
//	Conflicts — local skill differs from bundled (kept; never overwritten)
//	Failed    — write or read failures encountered for this profile
type SkillSyncProfileResult struct {
	Profile         string
	Added           int
	Unchanged       int
	Conflicts       int
	Failed          int
	AddedSkillNames []string
}

// SkillSyncRunner is the seam invoked by RunUpdateLifecycle after a
// successful pull. Returning a non-nil error marks the sync as failed but
// never fails the overall update (Hermes' best-effort `try/except: pass`
// contract). A nil seam in UpdateLifecycleOptions disables sync entirely
// and emits no evidence (silent default).
type SkillSyncRunner func(ctx context.Context) (SkillSyncResult, error)

type UpdateEvidence struct {
	Kind   UpdateEvidenceKind
	Detail string
}

type UpdateReport struct {
	Branch           string
	PreviousBranch   string
	Failed           bool
	Evidence         []UpdateEvidence
	OperatorRecovery string
	DashboardPIDs    []int
}

func (r *UpdateReport) add(kind UpdateEvidenceKind, detail string) {
	r.Evidence = append(r.Evidence, UpdateEvidence{Kind: kind, Detail: detail})
}

func (r *UpdateReport) append(other UpdateReport) {
	r.Failed = r.Failed || other.Failed
	r.Evidence = append(r.Evidence, other.Evidence...)
	if r.OperatorRecovery == "" {
		r.OperatorRecovery = other.OperatorRecovery
	}
	if len(other.DashboardPIDs) > 0 {
		r.DashboardPIDs = append(r.DashboardPIDs, other.DashboardPIDs...)
	}
}

type UpdateLifecycleOptions struct {
	CheckoutDir            string
	Branch                 string
	CheckOnly              bool
	Yes                    bool
	RestartGateway         string
	KillStaleDashboard     bool
	Git                    UpdateGitRunner
	GatewayRestartPoll     *ServiceRestartPollReport
	StaleDashboardPIDs     []int
	KillStaleDashboardFunc func(int) error
	// Backup mirrors the --backup CLI flag: opt-in to a single-run
	// pre-update backup. Resolved through ResolveBackupPolicy along with
	// NoBackup; --no-backup wins when both are set.
	Backup bool
	// NoBackup mirrors the --no-backup CLI flag and beats Backup.
	NoBackup bool
	// BackupConfigEnabled mirrors updates.pre_update_backup from config
	// (default false). Wiring this from real config is a follow-up slice;
	// for now the flag-driven path is the only enabled surface.
	BackupConfigEnabled bool
	// BackupWriter is the optional zip-creation seam. When policy resolves
	// to Requested AND this seam is non-nil, the lifecycle calls it and
	// emits update_pre_backup_completed (with path/size/duration in the
	// detail) or update_pre_backup_failed. A nil seam falls back to the
	// policy-only behavior shipped in 56efc4042.
	BackupWriter BackupWriter
	// SkillSync is the optional bundled-skill profile-sync seam. When set,
	// it runs after a successful pull and emits update_skill_sync_completed
	// or update_skill_sync_failed evidence. A nil seam disables sync
	// entirely (silent default — emits no skill_sync_* evidence).
	SkillSync SkillSyncRunner
	// WebBuild is the optional web UI rebuild seam. When set, it runs
	// after skill sync and emits update_web_build_{completed,skipped,
	// unavailable,failed} evidence. Best-effort: errors are reported but
	// do not fail the update. A nil seam disables the step entirely
	// (silent default — emits no web_build_* evidence).
	WebBuild WebBuildRunner
	// ConfigCheck is the optional config-version check seam. When set, it
	// runs after the web build and reports current vs. latest schema
	// versions. ConfigMigrate is invoked only when the operator passed
	// Yes=true AND the check reports an outdated config; otherwise the
	// lifecycle emits update_config_migrate_needed advisory evidence.
	ConfigCheck ConfigCheckRunner
	// ConfigMigrate is the optional auto-apply seam. Only invoked when
	// Yes=true and ConfigCheck reports an outdated version. Errors emit
	// update_config_migrate_failed but never fail the update.
	ConfigMigrate ConfigMigrateRunner
}

type UpdateGitRunner interface {
	RunGit(ctx context.Context, cwd string, args ...string) UpdateGitResult
}

type UpdateGitResult struct {
	Stdout string
	Stderr string
	Err    error
}

type RealUpdateGitRunner struct {
	Git string
}

func (r RealUpdateGitRunner) RunGit(ctx context.Context, cwd string, args ...string) UpdateGitResult {
	git := r.Git
	if git == "" {
		git = "git"
	}
	cmd := exec.CommandContext(ctx, git, args...)
	cmd.Dir = cwd
	out, err := cmd.Output()
	result := UpdateGitResult{Stdout: string(out), Err: err}
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			result.Stderr = string(exit.Stderr)
		}
	}
	return result
}

// emitSkillSync invokes the optional bundled-skill profile-sync seam after
// a successful pull and emits one evidence record per profile (success
// path) or one failure record (error path). A nil seam emits no evidence
// at all (silent default — most operators don't have a multi-profile
// setup and don't need to hear about a no-op on every update).
//
// Best-effort contract: a non-nil error from the seam never fails the
// overall update — it only logs `update_skill_sync_failed` evidence so
// operators can see what went wrong. This matches Hermes' upstream
// `try/except: pass` pattern for skill sync.
func emitSkillSync(ctx context.Context, report *UpdateReport, options UpdateLifecycleOptions) {
	if options.SkillSync == nil {
		return
	}
	result, err := options.SkillSync(ctx)
	if err != nil {
		report.add(UpdateEvidenceSkillSyncFailed, err.Error())
		return
	}
	for _, p := range result.Profiles {
		report.add(UpdateEvidenceSkillSyncCompleted, formatSkillSyncSummary(p))
	}
}

// formatSkillSyncSummary renders the count line in Hermes-parity shape:
//
//	`<profile>: +N new, K unchanged[, M user-modified][, F failed]`
//
// Zero-count buckets except `Added`+`Unchanged` are omitted to keep the
// transcript short on the common case (most updates touch nothing).
func formatSkillSyncSummary(p SkillSyncProfileResult) string {
	parts := []string{
		fmt.Sprintf("%s: +%d new", p.Profile, p.Added),
		fmt.Sprintf("%d unchanged", p.Unchanged),
	}
	if p.Conflicts > 0 {
		parts = append(parts, fmt.Sprintf("%d user-modified (kept)", p.Conflicts))
	}
	if p.Failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", p.Failed))
	}
	return strings.Join(parts, ", ")
}

// emitConfigMigrate inspects the on-disk config schema version through
// the ConfigCheck seam and either auto-applies a migration (when
// options.Yes is true and ConfigMigrate is non-nil) or emits an advisory
// `update_config_migrate_needed` record pointing the operator at the
// `gormes config migrate` command.
//
// Branch summary:
//
//	ConfigCheck nil                          → silent default (no evidence)
//	ConfigCheck err                          → update_config_migrate_failed
//	current >= latest                        → silent default (no nag)
//	current < latest, not Yes                → update_config_migrate_needed
//	current < latest, Yes, ConfigMigrate nil → update_config_migrate_needed
//	current < latest, Yes, migrate err       → update_config_migrate_failed
//	current < latest, Yes, migrate ok        → update_config_migrate_completed
//
// Best-effort: a non-nil error from either seam never fails the overall
// update.
func emitConfigMigrate(ctx context.Context, report *UpdateReport, options UpdateLifecycleOptions) {
	if options.ConfigCheck == nil {
		return
	}
	res, err := options.ConfigCheck(ctx)
	if err != nil {
		report.add(UpdateEvidenceConfigMigrateFailed, err.Error())
		return
	}
	if res.Current >= res.Latest {
		return
	}
	if !options.Yes || options.ConfigMigrate == nil {
		report.add(
			UpdateEvidenceConfigMigrateNeeded,
			fmt.Sprintf("config schema v%d → v%d available; run `gormes config migrate` or rerun with --yes", res.Current, res.Latest),
		)
		return
	}
	if err := options.ConfigMigrate(ctx); err != nil {
		report.add(UpdateEvidenceConfigMigrateFailed, err.Error())
		return
	}
	report.add(
		UpdateEvidenceConfigMigrateCompleted,
		fmt.Sprintf("config schema migrated v%d → v%d", res.Current, res.Latest),
	)
}

// emitWebBuild invokes the optional web UI rebuild seam after skill sync
// and maps the result to one of four typed evidence kinds:
//
//	completed   — successful rebuild (Detail used as evidence detail)
//	skipped     — operator policy or no package.json (Reason used)
//	unavailable — required toolchain missing (Reason used)
//	failed      — non-zero exit from the build (error message used)
//
// Best-effort contract: a non-nil error never fails the overall update.
// A nil seam disables the step and emits no evidence (silent default for
// non-managed checkouts and runtimes without a web/ tree).
func emitWebBuild(ctx context.Context, report *UpdateReport, options UpdateLifecycleOptions) {
	if options.WebBuild == nil {
		return
	}
	res, err := options.WebBuild(ctx)
	if err != nil {
		report.add(UpdateEvidenceWebBuildFailed, err.Error())
		return
	}
	switch {
	case res.Unavailable:
		report.add(UpdateEvidenceWebBuildUnavailable, res.Reason)
	case res.Skipped:
		report.add(UpdateEvidenceWebBuildSkipped, res.Reason)
	default:
		report.add(UpdateEvidenceWebBuildCompleted, res.Detail)
	}
}

// emitPreUpdateBackupPolicy resolves the operator-visible pre-update backup
// policy and appends a typed evidence record to report when the operator
// explicitly opted in or out (or when config has it enabled). The default
// path — neither --backup nor --no-backup nor BackupConfigEnabled — emits
// NO evidence so the structured progress UX stays quiet on the common case.
//
// When policy resolves to Requested:
//   - BackupWriter wired → invoke it and emit pre_backup_completed (with
//     path/size/duration) or pre_backup_failed (with error message).
//   - BackupWriter nil    → fall back to the policy-only behavior shipped
//     in 56efc4042: emit pre_backup_requested with a deferral note so
//     existing tests and operator transcripts don't regress.
//
// Best-effort: a writer error never sets report.Failed (matches Hermes'
// "never raises" contract for backup).
func emitPreUpdateBackupPolicy(ctx context.Context, report *UpdateReport, options UpdateLifecycleOptions) {
	if !options.Backup && !options.NoBackup && !options.BackupConfigEnabled {
		return
	}
	decision := ResolveBackupPolicy(BackupPolicyFlags{
		Backup:        options.Backup,
		NoBackup:      options.NoBackup,
		ConfigEnabled: options.BackupConfigEnabled,
	})
	if !decision.Requested {
		report.add(
			UpdateEvidencePreBackupSkipped,
			fmt.Sprintf("%s; ~/.gormes left untouched before update", decision.Reason),
		)
		return
	}
	if options.BackupWriter == nil {
		report.add(
			UpdateEvidencePreBackupRequested,
			fmt.Sprintf("%s; backup writer not yet implemented (planned next slice)", decision.Reason),
		)
		return
	}
	res, err := options.BackupWriter(ctx)
	if err != nil {
		report.add(UpdateEvidencePreBackupFailed, err.Error())
		return
	}
	detail := fmt.Sprintf(
		"%s (%s, %s, %d file%s)",
		res.Path,
		formatBackupSize(res.SizeBytes),
		formatBackupDuration(res.DurationMs),
		res.FileCount,
		pluralBackupSuffix(res.FileCount),
	)
	if res.PrunedCount > 0 {
		detail += fmt.Sprintf("; pruned %d older (%s freed)", res.PrunedCount, formatBackupSize(res.PrunedBytes))
	}
	report.add(UpdateEvidencePreBackupCompleted, detail)
}

// appendRollbackHintIfApplicable adds an `update_rollback_hint`
// evidence record to the report when:
//   - the lifecycle terminated with Failed=true, AND
//   - a `pre_backup_completed` record is in evidence (proof a usable
//     zip exists on disk).
//
// The hint names `gormes restore --latest --yes` so operators see the
// recovery command inline without having to look up the restore CLI
// shape from documentation. Skipped on success, on no-backup runs, and
// on backup-write failures (no zip = nothing to restore).
func appendRollbackHintIfApplicable(report *UpdateReport) {
	if report == nil || !report.Failed {
		return
	}
	hasCompletedBackup := false
	for _, ev := range report.Evidence {
		if ev.Kind == UpdateEvidencePreBackupCompleted {
			hasCompletedBackup = true
			break
		}
	}
	if !hasCompletedBackup {
		return
	}
	report.add(
		UpdateEvidenceRollbackHint,
		"to roll back to the pre-update state, run: gormes restore --latest --yes",
	)
}

// pluralBackupSuffix returns "s" when count != 1 so the human evidence
// detail reads naturally: "1 file" / "42 files".
func pluralBackupSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// formatBackupSize renders a byte count as a human-readable string with
// the smallest unit that keeps the number under 1024.
func formatBackupSize(bytes int64) string {
	const (
		kb = 1024
		mb = 1024 * 1024
		gb = 1024 * 1024 * 1024
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1fGB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1fMB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1fKB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// formatBackupDuration renders a millisecond count as either "Nms" (under
// 1s) or "N.NNs" (1s and above).
func formatBackupDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.2fs", float64(ms)/1000.0)
}

func RunUpdateLifecycle(ctx context.Context, options UpdateLifecycleOptions) (report UpdateReport) {
	// Append a rollback hint when the lifecycle ends with Failed=true
	// AND a usable pre-update zip exists on disk. Centralized in a
	// defer so every early-return failure path picks it up without
	// scattering the gating across 17 return points.
	defer func() { appendRollbackHintIfApplicable(&report) }()

	branch := strings.TrimSpace(options.Branch)
	if branch == "" {
		branch = "main"
	}
	checkoutDir := strings.TrimSpace(options.CheckoutDir)
	if checkoutDir == "" {
		checkoutDir = "."
	}
	report = UpdateReport{Branch: branch}

	if options.CheckOnly {
		report.add(UpdateEvidenceCheck, fmt.Sprintf("checked update readiness for %s", branch))
		return report
	}
	// Resolve pre-update backup policy BEFORE any git mutation. A backup
	// taken after the pull is useless for rollback. Silent default: when
	// neither --backup nor --no-backup is set and config has not opted in,
	// no evidence is emitted (matches Hermes' silent default — operators
	// don't need to hear about the skipped backup on every update run).
	emitPreUpdateBackupPolicy(ctx, &report, options)
	if options.Git == nil {
		report.Failed = true
		report.add(UpdateEvidenceNotManagedCheckout, "no git runner available")
		return report
	}

	if inside := options.Git.RunGit(ctx, checkoutDir, "rev-parse", "--is-inside-work-tree"); inside.Err != nil || strings.TrimSpace(inside.Stdout) != "true" {
		report.Failed = true
		report.add(UpdateEvidenceNotManagedCheckout, "checkout is not a managed git worktree")
		return report
	}

	head := options.Git.RunGit(ctx, checkoutDir, "rev-parse", "--abbrev-ref", "HEAD")
	currentBranch := strings.TrimSpace(head.Stdout)
	if currentBranch == "" {
		currentBranch = "HEAD"
	}
	report.PreviousBranch = currentBranch
	if currentBranch != branch {
		if result := options.Git.RunGit(ctx, checkoutDir, "checkout", branch); result.Err != nil {
			report.Failed = true
			report.add(UpdateEvidenceBranchSwitchFailed, gitFailureDetail(result))
			return report
		}
		if currentBranch == "HEAD" {
			report.add(UpdateEvidenceDetachedHeadSwitched, fmt.Sprintf("detached HEAD switched to %s", branch))
		} else {
			report.add(UpdateEvidenceBranchSwitched, fmt.Sprintf("%s switched to %s", currentBranch, branch))
		}
	}

	stashRef := ""
	if status := options.Git.RunGit(ctx, checkoutDir, "status", "--porcelain"); strings.TrimSpace(status.Stdout) != "" {
		if result := options.Git.RunGit(ctx, checkoutDir, "stash", "push", "--include-untracked", "-m", "gormes-update-autostash"); result.Err != nil {
			report.Failed = true
			report.add(UpdateEvidenceAutostashPreserved, gitFailureDetail(result))
			return report
		}
		ref := options.Git.RunGit(ctx, checkoutDir, "rev-parse", "--verify", "refs/stash")
		stashRef = strings.TrimSpace(ref.Stdout)
		if stashRef == "" || ref.Err != nil {
			report.Failed = true
			report.add(UpdateEvidenceAutostashPreserved, "could not resolve refs/stash after autostash")
			return report
		}
		report.add(UpdateEvidenceAutostashCreated, stashRef)
	}

	if result := options.Git.RunGit(ctx, checkoutDir, "fetch", "origin", branch); result.Err != nil {
		report.Failed = true
		report.add(classifyUpdateGitFailure(result), gitFailureDetail(result))
		preserveStash(&report, stashRef)
		return report
	}
	if result := options.Git.RunGit(ctx, checkoutDir, "pull", "--ff-only", "origin", branch); result.Err != nil {
		if kind := classifyUpdateGitFailure(result); kind == UpdateEvidenceNetworkError || kind == UpdateEvidenceAuthError {
			report.Failed = true
			report.add(kind, gitFailureDetail(result))
			preserveStash(&report, stashRef)
			return report
		}
		if reset := options.Git.RunGit(ctx, checkoutDir, "reset", "--hard", "origin/"+branch); reset.Err != nil {
			report.Failed = true
			report.add(UpdateEvidenceNetworkError, gitFailureDetail(reset))
			preserveStash(&report, stashRef)
			return report
		}
		report.add(UpdateEvidenceResetFallback, fmt.Sprintf("fast-forward pull failed; reset to origin/%s", branch))
	}

	if stashRef != "" {
		apply := options.Git.RunGit(ctx, checkoutDir, "stash", "apply", stashRef)
		if apply.Err != nil {
			report.Failed = true
			report.add(UpdateEvidenceAutostashPreserved, gitFailureDetail(apply))
			report.OperatorRecovery = stashRecovery(stashRef)
			return report
		}
		conflicts := options.Git.RunGit(ctx, checkoutDir, "diff", "--name-only", "--diff-filter=U")
		if strings.TrimSpace(conflicts.Stdout) != "" || conflicts.Err != nil {
			report.Failed = true
			report.add(UpdateEvidenceAutostashPreserved, "restore produced conflicts; stash preserved")
			report.OperatorRecovery = stashRecovery(stashRef)
			return report
		}
		if selector := resolveUpdateStashSelector(options.Git.RunGit(ctx, checkoutDir, "stash", "list", "--format=%gd %H").Stdout, stashRef); selector != "" {
			if drop := options.Git.RunGit(ctx, checkoutDir, "stash", "drop", selector); drop.Err != nil {
				report.add(UpdateEvidenceAutostashPreserved, gitFailureDetail(drop))
			}
		} else {
			report.add(UpdateEvidenceAutostashPreserved, "restored local changes but could not locate stash selector to drop")
		}
		report.add(UpdateEvidenceAutostashRestored, stashRef)
	}

	// Skill sync runs AFTER pull but BEFORE gateway restart so the gateway
	// can pick up newly-bundled skills on its restart cycle. Best-effort:
	// errors emit update_skill_sync_failed but never set report.Failed
	// (matches Hermes' `try/except: pass` for skill sync).
	emitSkillSync(ctx, &report, options)

	// Web UI rebuild runs after skill sync. Best-effort: build failures
	// emit update_web_build_failed but never set report.Failed
	// (matches Hermes' soft-failure contract — only `hermes web` treats
	// a build failure as fatal). Silent default when seam is nil.
	emitWebBuild(ctx, &report, options)

	// Config schema migration check runs after web build. When --yes is
	// set, the migration auto-applies; otherwise the lifecycle emits an
	// advisory `needed` evidence pointing the operator at
	// `gormes config migrate`. Silent default for already-current configs
	// and nil seams.
	emitConfigMigrate(ctx, &report, options)

	if options.GatewayRestartPoll != nil && strings.TrimSpace(options.RestartGateway) != "never" {
		report.append(EvaluateUpdateGatewayRestart(*options.GatewayRestartPoll))
	}
	report.append(HandleStaleDashboardProcesses(UpdateDashboardOptions{
		PIDs:     options.StaleDashboardPIDs,
		Kill:     options.KillStaleDashboard,
		KillFunc: options.KillStaleDashboardFunc,
	}))

	return report
}

func preserveStash(report *UpdateReport, stashRef string) {
	if stashRef == "" {
		return
	}
	report.add(UpdateEvidenceAutostashPreserved, "local changes remain in git stash")
	report.OperatorRecovery = stashRecovery(stashRef)
}

func stashRecovery(stashRef string) string {
	return "Restore manually with: git stash apply " + stashRef
}

func resolveUpdateStashSelector(stashList, stashRef string) string {
	for _, line := range strings.Split(stashList, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == stashRef {
			return fields[0]
		}
	}
	return ""
}

func classifyUpdateGitFailure(result UpdateGitResult) UpdateEvidenceKind {
	text := strings.ToLower(result.Stderr + "\n" + result.Stdout + "\n" + fmt.Sprint(result.Err))
	if strings.Contains(text, "auth") || strings.Contains(text, "permission denied") || strings.Contains(text, "403") || strings.Contains(text, "401") {
		return UpdateEvidenceAuthError
	}
	if strings.Contains(text, "could not resolve") ||
		strings.Contains(text, "failed to connect") ||
		strings.Contains(text, "connection") ||
		strings.Contains(text, "network") ||
		strings.Contains(text, "timed out") ||
		strings.Contains(text, "timeout") ||
		strings.Contains(text, "unable to access") ||
		strings.Contains(text, "repository not found") {
		return UpdateEvidenceNetworkError
	}
	return UpdateEvidenceGitError
}

func gitFailureDetail(result UpdateGitResult) string {
	if strings.TrimSpace(result.Stderr) != "" {
		return strings.TrimSpace(result.Stderr)
	}
	if strings.TrimSpace(result.Stdout) != "" {
		return strings.TrimSpace(result.Stdout)
	}
	if result.Err != nil {
		return result.Err.Error()
	}
	return "git command failed"
}

type UpdateHangupOptions struct {
	InstallSIGHUPIgnore func() (bool, error)
	LogAvailable        bool
}

type UpdateHangupReport struct {
	Evidence []UpdateEvidence
}

func InstallUpdateHangupProtection(options UpdateHangupOptions) UpdateHangupReport {
	var report UpdateHangupReport
	if options.InstallSIGHUPIgnore != nil {
		if ok, err := options.InstallSIGHUPIgnore(); ok && err == nil {
			report.Evidence = append(report.Evidence, UpdateEvidence{Kind: UpdateEvidenceHangupIgnored, Detail: "SIGHUP ignored"})
		} else {
			report.Evidence = append(report.Evidence, UpdateEvidence{Kind: UpdateEvidenceHangupUnavailable, Detail: fmt.Sprint(err)})
		}
	} else {
		report.Evidence = append(report.Evidence, UpdateEvidence{Kind: UpdateEvidenceHangupUnavailable, Detail: "SIGHUP unavailable"})
	}
	if options.LogAvailable {
		report.Evidence = append(report.Evidence, UpdateEvidence{Kind: UpdateEvidenceHangupLogMirrored, Detail: "output mirrored to update.log"})
	} else {
		report.Evidence = append(report.Evidence, UpdateEvidence{Kind: UpdateEvidenceHangupLogUnavailable, Detail: "update log unavailable"})
	}
	return report
}

type UpdateOutputMirror struct {
	original io.Writer
	log      io.Writer
}

func NewUpdateOutputMirror(original, log io.Writer) *UpdateOutputMirror {
	return &UpdateOutputMirror{original: original, log: log}
}

func (m *UpdateOutputMirror) Write(p []byte) (int, error) {
	if m.original != nil {
		_, _ = m.original.Write(p)
	}
	if m.log != nil {
		_, _ = m.log.Write(p)
	}
	return len(p), nil
}

func (m *UpdateOutputMirror) Flush() error {
	if f, ok := m.original.(interface{ Flush() error }); ok {
		_ = f.Flush()
	}
	if f, ok := m.log.(interface{ Flush() error }); ok {
		_ = f.Flush()
	}
	return nil
}

func EvaluateUpdateGatewayRestart(poll ServiceRestartPollReport) UpdateReport {
	report := UpdateReport{}
	switch poll.Outcome {
	case ServiceRestartPollRestarted:
		report.add(UpdateEvidenceGatewayRestarted, "gateway service reported active after restart")
	case ServiceRestartPollTimeout:
		report.Failed = true
		report.add(UpdateEvidenceGatewayRestartTimeout, "gateway restart timed out before active status")
	case ServiceRestartPollCrashedAfterRestart, ServiceRestartPollManagerUnavailable:
		report.Failed = true
		report.add(UpdateEvidenceGatewayRestartUnavailable, "gateway restart could not be validated")
	default:
		report.Failed = true
		report.add(UpdateEvidenceGatewayRestartUnavailable, "gateway restart outcome unavailable")
	}
	return report
}

type UpdateDashboardOptions struct {
	PIDs     []int
	Kill     bool
	KillFunc func(int) error
}

func HandleStaleDashboardProcesses(options UpdateDashboardOptions) UpdateReport {
	report := UpdateReport{DashboardPIDs: append([]int(nil), options.PIDs...)}
	if len(options.PIDs) == 0 {
		return report
	}
	report.add(UpdateEvidenceStaleDashboardDetected, fmt.Sprintf("%d dashboard process(es) detected", len(options.PIDs)))
	if !options.Kill {
		return report
	}
	for _, pid := range options.PIDs {
		if options.KillFunc != nil {
			if err := options.KillFunc(pid); err != nil {
				report.Failed = true
				report.add(UpdateEvidenceStaleDashboardKillFailed, fmt.Sprintf("pid %d: %v", pid, err))
				continue
			}
		}
		report.add(UpdateEvidenceStaleDashboardKilled, fmt.Sprintf("pid %d stopped", pid))
	}
	return report
}
