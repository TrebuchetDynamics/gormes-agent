package update

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

func TestUpdateCommandGatewayRestartPolicyControlsFailureSemantics(t *testing.T) {
	timeout := ServiceRestartPollReport{Outcome: ServiceRestartPollTimeout, Service: "gormes-gateway"}
	auto := EvaluateUpdateGatewayRestartPolicy("auto", &timeout)
	if auto.Failed {
		t.Fatalf("auto restart timeout should warn, not fail: %+v", auto)
	}
	assertUpdateEvidence(t, auto, UpdateEvidenceGatewayRestartTimeout)

	always := EvaluateUpdateGatewayRestartPolicy("always", &timeout)
	if !always.Failed {
		t.Fatalf("always restart timeout should fail: %+v", always)
	}
	assertUpdateEvidence(t, always, UpdateEvidenceGatewayRestartTimeout)

	never := EvaluateUpdateGatewayRestartPolicy("never", &timeout)
	if never.Failed {
		t.Fatalf("never restart should not fail: %+v", never)
	}
	assertUpdateEvidence(t, never, UpdateEvidenceGatewayRestartNeeded)
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

func TestUpdateCommandCheckFetchesRemoteAndReportsCommitDelta(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree":           {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":               {Stdout: "development\n"},
		"fetch origin development":                  {},
		"rev-parse HEAD":                            {Stdout: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n"},
		"rev-parse origin/development":              {Stdout: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n"},
		"rev-list --count HEAD..origin/development": {Stdout: "3\n"},
	})
	skillSyncCalled := false
	webBuildCalled := false
	configCheckCalled := false

	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir: "/repo/gormes",
		Branch:      "development",
		CheckOnly:   true,
		Git:         runner,
		SkillSync: func(context.Context) (SkillSyncResult, error) {
			skillSyncCalled = true
			return SkillSyncResult{}, nil
		},
		WebBuild: func(context.Context) (WebBuildResult, error) {
			webBuildCalled = true
			return WebBuildResult{}, nil
		},
		ConfigCheck: func(context.Context) (ConfigVersionResult, error) {
			configCheckCalled = true
			return ConfigVersionResult{}, nil
		},
	})

	if report.Failed {
		t.Fatalf("check lifecycle failed: %+v", report)
	}
	if report.PreviousBranch != "development" {
		t.Fatalf("PreviousBranch = %q, want development", report.PreviousBranch)
	}
	detail := findFirstEvidenceDetail(report, UpdateEvidenceCheckAvailable, "3 commit")
	if detail == "" {
		t.Fatalf("check report missing available-update evidence: %+v", report.Evidence)
	}
	for _, want := range []string{"origin/development", "aaaaaaaaaaaa", "bbbbbbbbbbbb"} {
		if !strings.Contains(detail, want) {
			t.Fatalf("available detail missing %q: %q", want, detail)
		}
	}
	if skillSyncCalled || webBuildCalled || configCheckCalled {
		t.Fatalf("check mode must not run post-update seams: skill=%v web=%v config=%v", skillSyncCalled, webBuildCalled, configCheckCalled)
	}
	assertUpdateGitCommands(t, runner,
		"rev-parse --is-inside-work-tree",
		"rev-parse --abbrev-ref HEAD",
		"fetch origin development",
		"rev-parse HEAD",
		"rev-parse origin/development",
		"rev-list --count HEAD..origin/development",
	)
}

func TestUpdateCommandPublishesBinaryBeforePostUpdateSteps(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree":   {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":       {Stdout: "development\n"},
		"status --porcelain":                {},
		"fetch origin development":          {},
		"pull --ff-only origin development": {},
	})
	var order []string

	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir: "/repo/gormes",
		Branch:      "development",
		Git:         runner,
		BinaryPublisher: func(_ context.Context, req UpdateBinaryPublishRequest) UpdateReport {
			order = append(order, "publish")
			if req.CheckoutDir != "/repo/gormes" || req.Branch != "development" {
				t.Fatalf("publish request = %+v, want checkout+branch", req)
			}
			return UpdateReport{Evidence: []UpdateEvidence{{Kind: UpdateEvidenceBuildCompleted, Detail: "built test binary"}}}
		},
		SkillSync: func(context.Context) (SkillSyncResult, error) {
			order = append(order, "skill")
			return SkillSyncResult{Profiles: []SkillSyncProfileResult{{Profile: "main"}}}, nil
		},
		WebBuild: func(context.Context) (WebBuildResult, error) {
			order = append(order, "web")
			return WebBuildResult{Detail: "web built"}, nil
		},
		ConfigCheck: func(context.Context) (ConfigVersionResult, error) {
			order = append(order, "config")
			return ConfigVersionResult{Current: 1, Latest: 1}, nil
		},
	})

	if report.Failed {
		t.Fatalf("RunUpdateLifecycle failed: %+v", report)
	}
	assertUpdateEvidence(t, report, UpdateEvidenceBuildCompleted)
	if !reflect.DeepEqual(order, []string{"publish", "skill", "web", "config"}) {
		t.Fatalf("order = %#v, want publish before post-update steps", order)
	}
}

func TestUpdateCommandPublishFailureStopsBeforePostUpdateSteps(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree":   {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":       {Stdout: "development\n"},
		"status --porcelain":                {},
		"fetch origin development":          {},
		"pull --ff-only origin development": {},
	})
	postUpdateCalled := false

	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir: "/repo/gormes",
		Branch:      "development",
		Git:         runner,
		BinaryPublisher: func(context.Context, UpdateBinaryPublishRequest) UpdateReport {
			return UpdateReport{
				Failed:   true,
				Evidence: []UpdateEvidence{{Kind: UpdateEvidencePublishFailed, Detail: "disk full"}},
			}
		},
		SkillSync: func(context.Context) (SkillSyncResult, error) {
			postUpdateCalled = true
			return SkillSyncResult{}, nil
		},
		WebBuild: func(context.Context) (WebBuildResult, error) {
			postUpdateCalled = true
			return WebBuildResult{}, nil
		},
		ConfigCheck: func(context.Context) (ConfigVersionResult, error) {
			postUpdateCalled = true
			return ConfigVersionResult{}, nil
		},
	})

	if !report.Failed {
		t.Fatalf("publish failure should fail lifecycle: %+v", report)
	}
	assertUpdateEvidence(t, report, UpdateEvidencePublishFailed)
	if postUpdateCalled {
		t.Fatalf("post-update steps must not run after failed binary publish")
	}
}

func TestUpdateCommandGatewayRestartRunnerRunsAfterPublisher(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree":   {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":       {Stdout: "development\n"},
		"status --porcelain":                {},
		"fetch origin development":          {},
		"pull --ff-only origin development": {},
	})
	var order []string

	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir:    "/repo/gormes",
		Branch:         "development",
		RestartGateway: "always",
		Git:            runner,
		BinaryPublisher: func(context.Context, UpdateBinaryPublishRequest) UpdateReport {
			order = append(order, "publish")
			return UpdateReport{Evidence: []UpdateEvidence{{Kind: UpdateEvidencePublishCompleted, Detail: "published"}}}
		},
		GatewayRestart: func(_ context.Context, req UpdateGatewayRestartRequest) UpdateReport {
			order = append(order, "restart")
			if req.Policy != "always" {
				t.Fatalf("restart policy = %q, want always", req.Policy)
			}
			return UpdateReport{Evidence: []UpdateEvidence{{Kind: UpdateEvidenceGatewayRestarted, Detail: "restarted"}}}
		},
	})

	if report.Failed {
		t.Fatalf("RunUpdateLifecycle failed: %+v", report)
	}
	if !reflect.DeepEqual(order, []string{"publish", "restart"}) {
		t.Fatalf("order = %#v, want publish then restart", order)
	}
	assertUpdateEvidence(t, report, UpdateEvidenceGatewayRestarted)
}

func TestUpdateCommandCheckReportsAlreadyCurrent(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree":    {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":        {Stdout: "main\n"},
		"fetch origin main":                  {},
		"rev-parse HEAD":                     {Stdout: "cccccccccccccccccccccccccccccccccccccccc\n"},
		"rev-parse origin/main":              {Stdout: "cccccccccccccccccccccccccccccccccccccccc\n"},
		"rev-list --count HEAD..origin/main": {Stdout: "0\n"},
	})

	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir: "/repo/gormes",
		Branch:      "main",
		CheckOnly:   true,
		Git:         runner,
	})

	if report.Failed {
		t.Fatalf("check lifecycle failed: %+v", report)
	}
	if detail := findFirstEvidenceDetail(report, UpdateEvidenceCheckCurrent, "already current"); detail == "" {
		t.Fatalf("check report missing current evidence: %+v", report.Evidence)
	}
}

func TestUpdateCommandCheckFetchFailureFailsWithoutMutation(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree": {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":     {Stdout: "main\n"},
		"fetch origin main":               {Stderr: "Could not resolve host: github.com", Err: errors.New("network")},
	})

	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir: "/repo/gormes",
		Branch:      "main",
		CheckOnly:   true,
		Git:         runner,
	})

	if !report.Failed {
		t.Fatalf("failed fetch should fail check report: %+v", report)
	}
	assertUpdateEvidence(t, report, UpdateEvidenceNetworkError)
	assertUpdateGitCommands(t, runner,
		"rev-parse --is-inside-work-tree",
		"rev-parse --abbrev-ref HEAD",
		"fetch origin main",
	)
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
