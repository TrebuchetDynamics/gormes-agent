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
)

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

// emitPreUpdateBackupPolicy resolves the operator-visible pre-update backup
// policy and appends a typed evidence record to report when the operator
// explicitly opted in or out (or when config has it enabled). The default
// path — neither --backup nor --no-backup nor BackupConfigEnabled — emits
// NO evidence so the structured progress UX stays quiet on the common case.
//
// The actual backup writer (zip creation, size/duration reporting, retention)
// is a follow-up slice. This row only surfaces the decision via a typed
// evidence record so downstream slices can wire the writer behind the same
// requested-evidence trigger.
func emitPreUpdateBackupPolicy(report *UpdateReport, options UpdateLifecycleOptions) {
	if !options.Backup && !options.NoBackup && !options.BackupConfigEnabled {
		return
	}
	decision := ResolveBackupPolicy(BackupPolicyFlags{
		Backup:        options.Backup,
		NoBackup:      options.NoBackup,
		ConfigEnabled: options.BackupConfigEnabled,
	})
	if decision.Requested {
		report.add(
			UpdateEvidencePreBackupRequested,
			fmt.Sprintf("%s; backup writer not yet implemented (planned next slice)", decision.Reason),
		)
		return
	}
	report.add(
		UpdateEvidencePreBackupSkipped,
		fmt.Sprintf("%s; ~/.gormes left untouched before update", decision.Reason),
	)
}

func RunUpdateLifecycle(ctx context.Context, options UpdateLifecycleOptions) UpdateReport {
	branch := strings.TrimSpace(options.Branch)
	if branch == "" {
		branch = "main"
	}
	checkoutDir := strings.TrimSpace(options.CheckoutDir)
	if checkoutDir == "" {
		checkoutDir = "."
	}
	report := UpdateReport{Branch: branch}

	if options.CheckOnly {
		report.add(UpdateEvidenceCheck, fmt.Sprintf("checked update readiness for %s", branch))
		return report
	}
	// Resolve pre-update backup policy BEFORE any git mutation. A backup
	// taken after the pull is useless for rollback. Silent default: when
	// neither --backup nor --no-backup is set and config has not opted in,
	// no evidence is emitted (matches Hermes' silent default — operators
	// don't need to hear about the skipped backup on every update run).
	emitPreUpdateBackupPolicy(&report, options)
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
