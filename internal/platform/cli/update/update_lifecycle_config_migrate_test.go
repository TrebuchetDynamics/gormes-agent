package update

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeConfigMigrate struct {
	checkVer   ConfigVersionResult
	checkErr   error
	migrateErr error
	checkCalls int
	applyCalls int
}

func (f *fakeConfigMigrate) Check(_ context.Context) (ConfigVersionResult, error) {
	f.checkCalls++
	return f.checkVer, f.checkErr
}

func (f *fakeConfigMigrate) Apply(_ context.Context) error {
	f.applyCalls++
	return f.migrateErr
}

// TestUpdateLifecycle_NilConfigCheck_EmitsNoEvidence proves the
// silent-default contract: when no ConfigCheck seam is wired (early
// callers, tests), no update_config_migrate_* evidence is emitted.
func TestUpdateLifecycle_NilConfigCheck_EmitsNoEvidence(t *testing.T) {
	report := runCleanLifecycle(t, UpdateLifecycleOptions{})
	for _, ev := range report.Evidence {
		if strings.HasPrefix(string(ev.Kind), "update_config_migrate_") {
			t.Fatalf("nil ConfigCheck must not emit update_config_migrate_*; got %q", ev.Kind)
		}
	}
}

// TestUpdateLifecycle_ConfigAlreadyCurrent_EmitsNoEvidence proves the
// no-op silent path: when the on-disk config is already at the latest
// version, no advisory evidence is emitted (operators don't need a
// migration nag on every update when nothing needs migrating).
func TestUpdateLifecycle_ConfigAlreadyCurrent_EmitsNoEvidence(t *testing.T) {
	cfg := &fakeConfigMigrate{checkVer: ConfigVersionResult{Current: 19, Latest: 19}}
	report := runCleanLifecycle(t, UpdateLifecycleOptions{ConfigCheck: cfg.Check, ConfigMigrate: cfg.Apply})
	for _, ev := range report.Evidence {
		if strings.HasPrefix(string(ev.Kind), "update_config_migrate_") {
			t.Fatalf("up-to-date config must not emit update_config_migrate_*; got %q", ev.Kind)
		}
	}
	if cfg.applyCalls != 0 {
		t.Fatalf("up-to-date config must not invoke ConfigMigrate seam; got %d calls", cfg.applyCalls)
	}
}

// TestUpdateLifecycle_ConfigOutdatedWithoutYes_EmitsNeededEvidence proves
// the advisory path: when migration is needed but --yes is NOT set, the
// lifecycle emits update_config_migrate_needed with version numbers but
// does NOT call the migrate seam (operator must opt-in explicitly).
func TestUpdateLifecycle_ConfigOutdatedWithoutYes_EmitsNeededEvidence(t *testing.T) {
	cfg := &fakeConfigMigrate{checkVer: ConfigVersionResult{Current: 11, Latest: 19}}
	report := runCleanLifecycle(t, UpdateLifecycleOptions{ConfigCheck: cfg.Check, ConfigMigrate: cfg.Apply})
	if cfg.applyCalls != 0 {
		t.Fatalf("without --yes, ConfigMigrate must NOT be called; got %d calls", cfg.applyCalls)
	}
	detail := findFirstEvidenceDetail(report, UpdateEvidenceConfigMigrateNeeded, "11")
	if detail == "" {
		t.Fatalf("update_config_migrate_needed must include the current version; got: %+v", report.Evidence)
	}
	if !strings.Contains(detail, "19") {
		t.Fatalf("update_config_migrate_needed must include the latest version; got %q", detail)
	}
	if !strings.Contains(detail, "gormes config migrate") {
		t.Fatalf("update_config_migrate_needed must guide the operator to the migrate command; got %q", detail)
	}
}

// TestUpdateLifecycle_ConfigOutdatedWithYes_AutoApplies proves the
// auto-apply path: --yes invokes the migrate seam exactly once and emits
// update_config_migrate_completed with version numbers.
func TestUpdateLifecycle_ConfigOutdatedWithYes_AutoApplies(t *testing.T) {
	cfg := &fakeConfigMigrate{checkVer: ConfigVersionResult{Current: 11, Latest: 19}}
	report := runCleanLifecycle(t, UpdateLifecycleOptions{
		ConfigCheck:   cfg.Check,
		ConfigMigrate: cfg.Apply,
		Yes:           true,
	})
	if cfg.applyCalls != 1 {
		t.Fatalf("--yes must call ConfigMigrate exactly once; got %d", cfg.applyCalls)
	}
	if findFirstEvidenceDetail(report, UpdateEvidenceConfigMigrateCompleted, "11") == "" {
		t.Fatalf("completed evidence must name from-version; got: %+v", report.Evidence)
	}
	if findFirstEvidenceDetail(report, UpdateEvidenceConfigMigrateCompleted, "19") == "" {
		t.Fatalf("completed evidence must name to-version; got: %+v", report.Evidence)
	}
}

// TestUpdateLifecycle_ConfigMigrateError_EmitsFailedContinues proves the
// best-effort contract: when the migrate seam returns an error, the
// update still completes successfully.
func TestUpdateLifecycle_ConfigMigrateError_EmitsFailedContinues(t *testing.T) {
	cfg := &fakeConfigMigrate{
		checkVer:   ConfigVersionResult{Current: 11, Latest: 19},
		migrateErr: errors.New("write config.toml: permission denied"),
	}
	report := runCleanLifecycle(t, UpdateLifecycleOptions{
		ConfigCheck:   cfg.Check,
		ConfigMigrate: cfg.Apply,
		Yes:           true,
	})
	if report.Failed {
		t.Fatalf("config migrate error must NOT fail the update; got: %+v", report)
	}
	if findFirstEvidenceDetail(report, UpdateEvidenceConfigMigrateFailed, "permission denied") == "" {
		t.Fatalf("update_config_migrate_failed must include the error message; got: %+v", report.Evidence)
	}
}

// TestUpdateLifecycle_ConfigCheckError_EmitsFailedContinues proves the
// check-side error path: a Check() failure (e.g., invalid TOML, newer
// binary) emits failed evidence but does not fail the update.
func TestUpdateLifecycle_ConfigCheckError_EmitsFailedContinues(t *testing.T) {
	cfg := &fakeConfigMigrate{
		checkErr: errors.New("config: invalid TOML"),
	}
	report := runCleanLifecycle(t, UpdateLifecycleOptions{
		ConfigCheck:   cfg.Check,
		ConfigMigrate: cfg.Apply,
	})
	if report.Failed {
		t.Fatalf("config check error must NOT fail the update; got: %+v", report)
	}
	if findFirstEvidenceDetail(report, UpdateEvidenceConfigMigrateFailed, "invalid TOML") == "" {
		t.Fatalf("update_config_migrate_failed must include the check-error message; got: %+v", report.Evidence)
	}
	if cfg.applyCalls != 0 {
		t.Fatalf("ConfigMigrate must NOT be called when ConfigCheck failed; got %d calls", cfg.applyCalls)
	}
}

// runCleanLifecycle is a tiny helper that runs RunUpdateLifecycle against
// a happy-path fake git runner so each test focuses on the new
// ConfigCheck/ConfigMigrate seam.
func runCleanLifecycle(t *testing.T, opts UpdateLifecycleOptions) UpdateReport {
	t.Helper()
	git := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree": {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":     {Stdout: "main\n"},
		"status --porcelain":              {},
		"fetch origin main":               {},
		"pull --ff-only origin main":      {},
	})
	if opts.CheckoutDir == "" {
		opts.CheckoutDir = "/repo/gormes"
	}
	if opts.Branch == "" {
		opts.Branch = "main"
	}
	opts.Git = git
	report := RunUpdateLifecycle(context.Background(), opts)
	if report.Failed {
		t.Fatalf("RunUpdateLifecycle unexpectedly failed: %+v", report)
	}
	return report
}
