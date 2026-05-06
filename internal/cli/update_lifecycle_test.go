package cli

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestUpdateCommandAutostashDirtyCheckout(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree":   {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":       {Stdout: "development\n"},
		"status --porcelain":                {Stdout: " M cmd/gormes/main.go\n?? notes.txt\n"},
		"rev-parse --verify refs/stash":     {Stdout: "stash-commit-abc\n"},
		"fetch origin development":          {},
		"pull --ff-only origin development": {},
		"stash apply stash-commit-abc":      {},
		"stash list --format=%gd %H":        {Stdout: "stash@{0} stash-commit-abc\n"},
		"stash drop stash@{0}":              {},
	})

	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir: "/repo/gormes",
		Branch:      "development",
		Git:         runner,
	})

	if report.Failed {
		t.Fatalf("RunUpdateLifecycle failed: %+v", report)
	}
	assertUpdateEvidence(t, report, UpdateEvidenceAutostashCreated)
	assertUpdateEvidence(t, report, UpdateEvidenceAutostashRestored)
	assertUpdateGitCommands(t, runner,
		"rev-parse --is-inside-work-tree",
		"rev-parse --abbrev-ref HEAD",
		"status --porcelain",
		"stash push --include-untracked -m gormes-update-autostash",
		"rev-parse --verify refs/stash",
		"fetch origin development",
		"pull --ff-only origin development",
		"stash apply stash-commit-abc",
		"diff --name-only --diff-filter=U",
		"stash list --format=%gd %H",
		"stash drop stash@{0}",
	)
}

func TestUpdateCommandAutostashRestoreConflictPreservesStash(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree":   {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":       {Stdout: "development\n"},
		"status --porcelain":                {Stdout: " M internal/cli/update_lifecycle.go\n"},
		"rev-parse --verify refs/stash":     {Stdout: "stash-commit-conflict\n"},
		"fetch origin development":          {},
		"pull --ff-only origin development": {},
		"stash apply stash-commit-conflict": {Stderr: "CONFLICT\n", Err: errors.New("conflict")},
	})

	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir: "/repo/gormes",
		Branch:      "development",
		Git:         runner,
	})

	if !report.Failed {
		t.Fatalf("RunUpdateLifecycle conflict Failed=false; report=%+v", report)
	}
	assertUpdateEvidence(t, report, UpdateEvidenceAutostashPreserved)
	if !strings.Contains(report.OperatorRecovery, "git stash apply stash-commit-conflict") {
		t.Fatalf("OperatorRecovery = %q, want manual stash apply guidance", report.OperatorRecovery)
	}
}

func TestUpdateCommandBranchAndDetachedHeadRecovery(t *testing.T) {
	for _, tc := range []struct {
		name       string
		head       string
		wantPrev   string
		wantStatus UpdateEvidenceKind
	}{
		{name: "feature branch", head: "feature/demo\n", wantPrev: "feature/demo", wantStatus: UpdateEvidenceBranchSwitched},
		{name: "detached head", head: "HEAD\n", wantPrev: "HEAD", wantStatus: UpdateEvidenceDetachedHeadSwitched},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
				"rev-parse --is-inside-work-tree":   {Stdout: "true\n"},
				"rev-parse --abbrev-ref HEAD":       {Stdout: tc.head},
				"checkout development":              {},
				"status --porcelain":                {},
				"fetch origin development":          {},
				"pull --ff-only origin development": {},
			})

			report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
				CheckoutDir: "/repo/gormes",
				Branch:      "development",
				Git:         runner,
			})

			if report.Failed {
				t.Fatalf("RunUpdateLifecycle failed: %+v", report)
			}
			if report.PreviousBranch != tc.wantPrev {
				t.Fatalf("PreviousBranch = %q, want %q", report.PreviousBranch, tc.wantPrev)
			}
			assertUpdateEvidence(t, report, tc.wantStatus)
			assertUpdateGitCommands(t, runner,
				"rev-parse --is-inside-work-tree",
				"rev-parse --abbrev-ref HEAD",
				"checkout development",
				"status --porcelain",
				"fetch origin development",
				"pull --ff-only origin development",
			)
		})
	}
}

func TestUpdateCommandNetworkAndAuthErrorsRenderFriendlyFailures(t *testing.T) {
	for _, tc := range []struct {
		name     string
		result   UpdateGitResult
		evidence UpdateEvidenceKind
	}{
		{
			name:     "network",
			result:   UpdateGitResult{Stderr: "Could not resolve host: github.com", Err: errors.New("network")},
			evidence: UpdateEvidenceNetworkError,
		},
		{
			name:     "auth",
			result:   UpdateGitResult{Stderr: "Authentication failed for 'https://example.invalid/repo.git'", Err: errors.New("auth")},
			evidence: UpdateEvidenceAuthError,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
				"rev-parse --is-inside-work-tree": {Stdout: "true\n"},
				"rev-parse --abbrev-ref HEAD":     {Stdout: "development\n"},
				"status --porcelain":              {Stdout: " M local.txt\n"},
				"rev-parse --verify refs/stash":   {Stdout: "stash-commit-safe\n"},
				"fetch origin development":        tc.result,
			})

			report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
				CheckoutDir: "/repo/gormes",
				Branch:      "development",
				Git:         runner,
			})

			if !report.Failed {
				t.Fatalf("RunUpdateLifecycle Failed=false; report=%+v", report)
			}
			assertUpdateEvidence(t, report, tc.evidence)
			assertUpdateEvidence(t, report, UpdateEvidenceAutostashPreserved)
			if !strings.Contains(report.OperatorRecovery, "git stash apply stash-commit-safe") {
				t.Fatalf("OperatorRecovery = %q, want stash recovery guidance", report.OperatorRecovery)
			}
		})
	}
}

func TestUpdateCommandHangupProtectionMirrorsOutput(t *testing.T) {
	var terminal, log strings.Builder
	mirror := NewUpdateOutputMirror(&terminal, &log)

	if _, err := mirror.Write([]byte("fetching\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := mirror.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	report := InstallUpdateHangupProtection(UpdateHangupOptions{
		InstallSIGHUPIgnore: func() (bool, error) { return true, nil },
		LogAvailable:        true,
	})

	if terminal.String() != "fetching\n" || log.String() != "fetching\n" {
		t.Fatalf("mirror terminal=%q log=%q, want both fetching", terminal.String(), log.String())
	}
	assertHangupEvidence(t, report, UpdateEvidenceHangupIgnored)
	assertHangupEvidence(t, report, UpdateEvidenceHangupLogMirrored)
}

func TestUpdateCommandGatewayRestartUsesValidatedServiceSeam(t *testing.T) {
	for _, tc := range []struct {
		name     string
		poll     ServiceRestartPollReport
		evidence UpdateEvidenceKind
		failed   bool
	}{
		{
			name:     "restarted",
			poll:     ServiceRestartPollReport{Outcome: ServiceRestartPollRestarted, Service: "gormes-gateway"},
			evidence: UpdateEvidenceGatewayRestarted,
		},
		{
			name:     "timeout",
			poll:     ServiceRestartPollReport{Outcome: ServiceRestartPollTimeout, Service: "gormes-gateway"},
			evidence: UpdateEvidenceGatewayRestartTimeout,
			failed:   true,
		},
		{
			name:     "crashed",
			poll:     ServiceRestartPollReport{Outcome: ServiceRestartPollCrashedAfterRestart, Service: "gormes-gateway"},
			evidence: UpdateEvidenceGatewayRestartUnavailable,
			failed:   true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := EvaluateUpdateGatewayRestart(tc.poll)
			if report.Failed != tc.failed {
				t.Fatalf("Failed = %v, want %v (%+v)", report.Failed, tc.failed, report)
			}
			assertUpdateEvidence(t, report, tc.evidence)
		})
	}
}

func TestUpdateCommandStaleDashboardWarning(t *testing.T) {
	report := HandleStaleDashboardProcesses(UpdateDashboardOptions{
		PIDs: []int{12345, 12346},
		Kill: false,
	})
	if report.Failed {
		t.Fatalf("warn-only dashboard report failed: %+v", report)
	}
	assertUpdateEvidence(t, report, UpdateEvidenceStaleDashboardDetected)
	if len(report.DashboardPIDs) != 2 {
		t.Fatalf("DashboardPIDs = %v, want two pids", report.DashboardPIDs)
	}

	killed := HandleStaleDashboardProcesses(UpdateDashboardOptions{
		PIDs: []int{12345, 12346},
		Kill: true,
		KillFunc: func(pid int) error {
			if pid == 12346 {
				return errors.New("still running")
			}
			return nil
		},
	})
	assertUpdateEvidence(t, killed, UpdateEvidenceStaleDashboardKillFailed)
	if !killed.Failed {
		t.Fatalf("Kill failure Failed=false; report=%+v", killed)
	}
}

func TestUpdateCommandNotManagedCheckout(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree": {Stdout: "false\n", Err: errors.New("not a git checkout")},
	})
	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir: "/tmp/not-gormes",
		Branch:      "development",
		Git:         runner,
	})
	if !report.Failed {
		t.Fatalf("not-managed checkout Failed=false; report=%+v", report)
	}
	assertUpdateEvidence(t, report, UpdateEvidenceNotManagedCheckout)
}

type fakeUpdateGitRunner struct {
	results  map[string]UpdateGitResult
	commands []string
}

func newFakeUpdateGitRunner(results map[string]UpdateGitResult) *fakeUpdateGitRunner {
	return &fakeUpdateGitRunner{results: results}
}

func (r *fakeUpdateGitRunner) RunGit(_ context.Context, _ string, args ...string) UpdateGitResult {
	key := strings.Join(args, " ")
	r.commands = append(r.commands, key)
	if result, ok := r.results[key]; ok {
		return result
	}
	return UpdateGitResult{}
}

func assertUpdateGitCommands(t *testing.T, runner *fakeUpdateGitRunner, want ...string) {
	t.Helper()
	if !reflect.DeepEqual(runner.commands, want) {
		t.Fatalf("git commands = %#v\nwant %#v", runner.commands, want)
	}
}

func assertUpdateEvidence(t *testing.T, report UpdateReport, kind UpdateEvidenceKind) {
	t.Helper()
	for _, evidence := range report.Evidence {
		if evidence.Kind == kind {
			return
		}
	}
	t.Fatalf("missing evidence %q in %+v", kind, report.Evidence)
}

func assertHangupEvidence(t *testing.T, report UpdateHangupReport, kind UpdateEvidenceKind) {
	t.Helper()
	for _, evidence := range report.Evidence {
		if evidence.Kind == kind {
			return
		}
	}
	t.Fatalf("missing hangup evidence %q in %+v", kind, report.Evidence)
}

func TestUpdateCommandPullFallbackResetEvidence(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree":   {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":       {Stdout: "development\n"},
		"status --porcelain":                {},
		"fetch origin development":          {},
		"pull --ff-only origin development": {Stderr: "Not possible to fast-forward", Err: errors.New("non-ff")},
		"reset --hard origin/development":   {},
	})

	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir: "/repo/gormes",
		Branch:      "development",
		Git:         runner,
	})

	if report.Failed {
		t.Fatalf("RunUpdateLifecycle reset fallback failed: %+v", report)
	}
	assertUpdateEvidence(t, report, UpdateEvidenceResetFallback)
}

var _ = time.Second
