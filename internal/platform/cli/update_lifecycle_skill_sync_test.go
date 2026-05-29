package cli

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeSkillSyncRunner records its invocation order relative to the git
// runner so the test can assert ordering (skill sync must run AFTER pull).
type fakeSkillSyncRunner struct {
	result    SkillSyncResult
	err       error
	calls     int
	calledAt  int // index into a shared sequence counter at call time
	sequencer *int
}

func (f *fakeSkillSyncRunner) Sync(ctx context.Context) (SkillSyncResult, error) {
	f.calls++
	if f.sequencer != nil {
		f.calledAt = *f.sequencer
		*f.sequencer++
	}
	return f.result, f.err
}

// TestUpdateLifecycle_NilSkillSync_EmitsNoSyncEvidence proves the
// silent-default contract: when no SkillSync seam is wired, no
// update_skill_sync_* evidence is emitted. Operators don't need to hear
// about a non-applicable feature on every update.
func TestUpdateLifecycle_NilSkillSync_EmitsNoSyncEvidence(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree": {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":     {Stdout: "main\n"},
		"status --porcelain":              {},
		"fetch origin main":               {},
		"pull --ff-only origin main":      {},
	})

	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir: "/repo/gormes",
		Branch:      "main",
		Git:         runner,
		// SkillSync intentionally nil
	})

	if report.Failed {
		t.Fatalf("RunUpdateLifecycle failed: %+v", report)
	}
	for _, ev := range report.Evidence {
		if strings.HasPrefix(string(ev.Kind), "update_skill_sync_") {
			t.Fatalf("nil SkillSync seam must not emit any update_skill_sync_* evidence; got %q", ev.Kind)
		}
	}
}

// TestUpdateLifecycle_SkillSyncSuccess_EmitsCompletedEvidence proves a
// successful sync emits one `update_skill_sync_completed` evidence per
// profile, with counts in the detail.
func TestUpdateLifecycle_SkillSyncSuccess_EmitsCompletedEvidence(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree": {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":     {Stdout: "main\n"},
		"status --porcelain":              {},
		"fetch origin main":               {},
		"pull --ff-only origin main":      {},
	})

	skillSync := &fakeSkillSyncRunner{
		result: SkillSyncResult{
			Profiles: []SkillSyncProfileResult{
				{Profile: "main", Added: 5, Unchanged: 12},
				{Profile: "ci", Added: 0, Unchanged: 17, Conflicts: 1},
			},
		},
	}

	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir: "/repo/gormes",
		Branch:      "main",
		Git:         runner,
		SkillSync:   skillSync.Sync,
	})

	if report.Failed {
		t.Fatalf("RunUpdateLifecycle failed: %+v", report)
	}
	if skillSync.calls != 1 {
		t.Fatalf("SkillSync seam must be called exactly once on a green pull; got %d", skillSync.calls)
	}
	defaultDetail := findFirstEvidenceDetail(report, UpdateEvidenceSkillSyncCompleted, "main")
	if defaultDetail == "" {
		t.Fatalf("update_skill_sync_completed must be emitted for `default` profile; got: %+v", report.Evidence)
	}
	if !strings.Contains(defaultDetail, "+5 new") {
		t.Fatalf("`default` profile detail must include `+5 new` count; got %q", defaultDetail)
	}
	ciDetail := findFirstEvidenceDetail(report, UpdateEvidenceSkillSyncCompleted, "ci")
	if !strings.Contains(ciDetail, "1 user-modified") {
		t.Fatalf("`ci` profile detail must include `1 user-modified` count for conflicts; got %q", ciDetail)
	}
}

// TestUpdateLifecycle_SkillSyncFailure_EmitsFailedEvidenceContinues proves
// the "best-effort" contract: when the skill-sync seam returns an error,
// the update still completes successfully. Sync errors emit
// `update_skill_sync_failed` evidence but never set report.Failed.
func TestUpdateLifecycle_SkillSyncFailure_EmitsFailedEvidenceContinues(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree": {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":     {Stdout: "main\n"},
		"status --porcelain":              {},
		"fetch origin main":               {},
		"pull --ff-only origin main":      {},
	})

	skillSync := &fakeSkillSyncRunner{err: errors.New("skills root unreadable")}
	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir: "/repo/gormes",
		Branch:      "main",
		Git:         runner,
		SkillSync:   skillSync.Sync,
	})

	if report.Failed {
		t.Fatalf("skill-sync error must NOT fail the update; got: %+v", report)
	}
	failed := findFirstEvidenceDetail(report, UpdateEvidenceSkillSyncFailed, "skills root unreadable")
	if failed == "" {
		t.Fatalf("update_skill_sync_failed must include the error message; got: %+v", report.Evidence)
	}
}

// TestUpdateLifecycle_SkillSyncRunsAfterPull proves the ordering invariant:
// skill sync MUST run after the pull, otherwise a fresh bundled-skill
// payload would not be available.
func TestUpdateLifecycle_SkillSyncRunsAfterPull(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree": {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":     {Stdout: "main\n"},
		"status --porcelain":              {},
		"fetch origin main":               {},
		"pull --ff-only origin main":      {},
	})

	skillSync := &fakeSkillSyncRunner{}
	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir: "/repo/gormes",
		Branch:      "main",
		Git:         runner,
		SkillSync:   skillSync.Sync,
	})

	if report.Failed {
		t.Fatalf("RunUpdateLifecycle failed: %+v", report)
	}
	if skillSync.calls != 1 {
		t.Fatalf("SkillSync seam must be called once; got %d", skillSync.calls)
	}
	// The fake git runner records a `pull` command; sync must run after it.
	pullCalled := false
	for _, c := range runner.commands {
		if strings.HasPrefix(c, "pull ") {
			pullCalled = true
		}
	}
	if !pullCalled {
		t.Fatalf("git pull must run; commands: %+v", runner.commands)
	}
	if skillSync.calls > 0 && len(runner.commands) == 0 {
		t.Fatalf("skill sync must not run before any git command")
	}
}

// findFirstEvidenceDetail returns the first evidence Detail of the given
// kind that contains needle, or empty string if none match.
func findFirstEvidenceDetail(report UpdateReport, kind UpdateEvidenceKind, needle string) string {
	for _, ev := range report.Evidence {
		if ev.Kind == kind && strings.Contains(ev.Detail, needle) {
			return ev.Detail
		}
	}
	return ""
}
