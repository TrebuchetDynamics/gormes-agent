package update

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeWebBuildRunner struct {
	result WebBuildResult
	err    error
	calls  int
}

func (f *fakeWebBuildRunner) Build(ctx context.Context) (WebBuildResult, error) {
	f.calls++
	return f.result, f.err
}

// TestUpdateLifecycle_NilWebBuild_EmitsNoEvidence proves the silent-default
// contract: when no WebBuild seam is wired, no update_web_build_* evidence
// is emitted. Most operators don't have a web/ tree and don't need to hear
// about a non-applicable feature on every update.
func TestUpdateLifecycle_NilWebBuild_EmitsNoEvidence(t *testing.T) {
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
	})

	if report.Failed {
		t.Fatalf("RunUpdateLifecycle failed: %+v", report)
	}
	for _, ev := range report.Evidence {
		if strings.HasPrefix(string(ev.Kind), "update_web_build_") {
			t.Fatalf("nil WebBuild seam must not emit update_web_build_*; got %q", ev.Kind)
		}
	}
}

// TestUpdateLifecycle_WebBuildCompleted_EmitsCompletedEvidence proves a
// successful run emits update_web_build_completed with the seam-supplied
// detail.
func TestUpdateLifecycle_WebBuildCompleted_EmitsCompletedEvidence(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree": {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":     {Stdout: "main\n"},
		"status --porcelain":              {},
		"fetch origin main":               {},
		"pull --ff-only origin main":      {},
	})

	web := &fakeWebBuildRunner{result: WebBuildResult{Detail: "web UI built in web/"}}
	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir: "/repo/gormes",
		Branch:      "main",
		Git:         runner,
		WebBuild:    web.Build,
	})

	if report.Failed {
		t.Fatalf("RunUpdateLifecycle failed: %+v", report)
	}
	if web.calls != 1 {
		t.Fatalf("WebBuild seam must be called exactly once; got %d", web.calls)
	}
	if findFirstEvidenceDetail(report, UpdateEvidenceWebBuildCompleted, "web UI built") == "" {
		t.Fatalf("update_web_build_completed must include seam Detail; got: %+v", report.Evidence)
	}
}

// TestUpdateLifecycle_WebBuildSkipped_EmitsSkippedEvidence proves the seam
// can declare an intentional skip (--skip-web, no package.json) without
// failing the update. Skipped reason surfaces in evidence detail.
func TestUpdateLifecycle_WebBuildSkipped_EmitsSkippedEvidence(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree": {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":     {Stdout: "main\n"},
		"status --porcelain":              {},
		"fetch origin main":               {},
		"pull --ff-only origin main":      {},
	})

	web := &fakeWebBuildRunner{result: WebBuildResult{Skipped: true, Reason: "--skip-web flag"}}
	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir: "/repo/gormes",
		Branch:      "main",
		Git:         runner,
		WebBuild:    web.Build,
	})

	if report.Failed {
		t.Fatalf("RunUpdateLifecycle failed: %+v", report)
	}
	if findFirstEvidenceDetail(report, UpdateEvidenceWebBuildSkipped, "--skip-web") == "" {
		t.Fatalf("update_web_build_skipped must include the seam Reason; got: %+v", report.Evidence)
	}
}

// TestUpdateLifecycle_WebBuildUnavailable_EmitsUnavailableEvidence proves
// the seam can report a missing toolchain (npm not on PATH) without failing
// the update. Unavailable is distinct from Skipped: skipped is a policy
// choice, unavailable is an environment limitation.
func TestUpdateLifecycle_WebBuildUnavailable_EmitsUnavailableEvidence(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree": {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":     {Stdout: "main\n"},
		"status --porcelain":              {},
		"fetch origin main":               {},
		"pull --ff-only origin main":      {},
	})

	web := &fakeWebBuildRunner{result: WebBuildResult{Unavailable: true, Reason: "npm not on PATH"}}
	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir: "/repo/gormes",
		Branch:      "main",
		Git:         runner,
		WebBuild:    web.Build,
	})

	if report.Failed {
		t.Fatalf("RunUpdateLifecycle failed: %+v", report)
	}
	if findFirstEvidenceDetail(report, UpdateEvidenceWebBuildUnavailable, "npm not on PATH") == "" {
		t.Fatalf("update_web_build_unavailable must include the seam Reason; got: %+v", report.Evidence)
	}
}

// TestUpdateLifecycle_WebBuildFailure_EmitsFailedEvidenceContinues proves
// the best-effort contract: when the seam returns an error, the update
// still completes successfully but evidence records the failure.
func TestUpdateLifecycle_WebBuildFailure_EmitsFailedEvidenceContinues(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree": {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":     {Stdout: "main\n"},
		"status --porcelain":              {},
		"fetch origin main":               {},
		"pull --ff-only origin main":      {},
	})

	web := &fakeWebBuildRunner{err: errors.New("npm run build exit 1")}
	report := RunUpdateLifecycle(context.Background(), UpdateLifecycleOptions{
		CheckoutDir: "/repo/gormes",
		Branch:      "main",
		Git:         runner,
		WebBuild:    web.Build,
	})

	if report.Failed {
		t.Fatalf("web-build error must NOT fail the update; got: %+v", report)
	}
	if findFirstEvidenceDetail(report, UpdateEvidenceWebBuildFailed, "exit 1") == "" {
		t.Fatalf("update_web_build_failed must include the error message; got: %+v", report.Evidence)
	}
}
