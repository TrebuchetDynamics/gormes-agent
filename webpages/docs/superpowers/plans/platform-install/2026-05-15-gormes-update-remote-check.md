# Gormes Update Remote Check Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Make `gormes update --check` fetch the configured remote branch, report exact commit delta, and avoid checkout/stash/pull/build/publish mutations.

**Architecture:** Keep the behavior in `internal/cli.RunUpdateLifecycle` because update already centralizes git and evidence handling there. The command layer continues to pass `CheckOnly`; JSON and human output use the existing `UpdateReport` evidence path.

**Tech Stack:** Go, Cobra command tests, fake git runner, `go test`.

---

## File Structure

- Modify `internal/cli/update_lifecycle.go`
  - Add `update_check_available` and `update_check_current` evidence kinds.
  - Replace the current `CheckOnly` early return with a real git-backed remote check.
  - Keep all check-mode behavior read-only except `git fetch`.
- Modify `internal/cli/update_lifecycle_test.go`
  - Add focused tests proving remote check behavior, no mutation commands, current-state reporting, and fetch failure classification.
- No `cmd/gormes` change is required for Slice 1 because existing JSON/human report rendering already prints lifecycle evidence.

## Task 1: Add RED Tests For Remote Check

**Files:**
- Modify: `internal/cli/update_lifecycle_test.go`

- [x] **Step 1: Add failing tests**

Append these tests near the existing update lifecycle tests:

```go
func TestUpdateCommandCheckFetchesRemoteAndReportsCommitDelta(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree":     {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":         {Stdout: "development\n"},
		"fetch origin development":            {},
		"rev-parse HEAD":                      {Stdout: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n"},
		"rev-parse origin/development":        {Stdout: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n"},
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

func TestUpdateCommandCheckReportsAlreadyCurrent(t *testing.T) {
	runner := newFakeUpdateGitRunner(map[string]UpdateGitResult{
		"rev-parse --is-inside-work-tree":     {Stdout: "true\n"},
		"rev-parse --abbrev-ref HEAD":         {Stdout: "main\n"},
		"fetch origin main":                   {},
		"rev-parse HEAD":                      {Stdout: "cccccccccccccccccccccccccccccccccccccccc\n"},
		"rev-parse origin/main":               {Stdout: "cccccccccccccccccccccccccccccccccccccccc\n"},
		"rev-list --count HEAD..origin/main":  {Stdout: "0\n"},
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
```

- [x] **Step 2: Run RED tests**

Run:

```sh
go test ./internal/cli -run 'TestUpdateCommandCheck(FetchesRemote|ReportsAlready|FetchFailure)' -count=1
```

Expected: FAIL because `UpdateEvidenceCheckAvailable` and `UpdateEvidenceCheckCurrent` do not exist yet, or because check mode returns before running git.

## Task 2: Implement Remote Check

**Files:**
- Modify: `internal/cli/update_lifecycle.go`

- [x] **Step 1: Add evidence constants**

Add these constants after `UpdateEvidenceCheck`:

```go
UpdateEvidenceCheckAvailable UpdateEvidenceKind = "update_check_available"
UpdateEvidenceCheckCurrent   UpdateEvidenceKind = "update_check_current"
```

- [x] **Step 2: Add imports**

Add `strconv` to the import block:

```go
import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)
```

- [x] **Step 3: Replace check-only early return**

Replace the existing `if options.CheckOnly { ... }` block in `RunUpdateLifecycle` with:

```go
if options.CheckOnly {
	runUpdateRemoteCheck(ctx, &report, options, checkoutDir, branch)
	return report
}
```

- [x] **Step 4: Add helper functions**

Add these helpers near `RunUpdateLifecycle`:

```go
func runUpdateRemoteCheck(ctx context.Context, report *UpdateReport, options UpdateLifecycleOptions, checkoutDir, branch string) {
	report.add(UpdateEvidenceCheck, fmt.Sprintf("checking remote update state for %s", branch))
	if options.Git == nil {
		report.Failed = true
		report.add(UpdateEvidenceNotManagedCheckout, "no git runner available")
		return
	}
	if inside := options.Git.RunGit(ctx, checkoutDir, "rev-parse", "--is-inside-work-tree"); inside.Err != nil || strings.TrimSpace(inside.Stdout) != "true" {
		report.Failed = true
		report.add(UpdateEvidenceNotManagedCheckout, "checkout is not a managed git worktree")
		return
	}
	head := options.Git.RunGit(ctx, checkoutDir, "rev-parse", "--abbrev-ref", "HEAD")
	report.PreviousBranch = strings.TrimSpace(head.Stdout)
	if report.PreviousBranch == "" {
		report.PreviousBranch = "HEAD"
	}
	if result := options.Git.RunGit(ctx, checkoutDir, "fetch", "origin", branch); result.Err != nil {
		report.Failed = true
		report.add(classifyUpdateGitFailure(result), gitFailureDetail(result))
		return
	}
	current := strings.TrimSpace(options.Git.RunGit(ctx, checkoutDir, "rev-parse", "HEAD").Stdout)
	remoteRef := "origin/" + branch
	remote := strings.TrimSpace(options.Git.RunGit(ctx, checkoutDir, "rev-parse", remoteRef).Stdout)
	countResult := options.Git.RunGit(ctx, checkoutDir, "rev-list", "--count", "HEAD.."+remoteRef)
	if countResult.Err != nil {
		report.Failed = true
		report.add(classifyUpdateGitFailure(countResult), gitFailureDetail(countResult))
		return
	}
	behind, err := strconv.Atoi(strings.TrimSpace(countResult.Stdout))
	if err != nil {
		report.Failed = true
		report.add(UpdateEvidenceGitError, fmt.Sprintf("parse remote commit count: %v", err))
		return
	}
	if behind > 0 {
		report.add(UpdateEvidenceCheckAvailable, fmt.Sprintf("%d commit%s behind %s (%s..%s)", behind, pluralCommitSuffix(behind), remoteRef, shortCommit(current), shortCommit(remote)))
		return
	}
	report.add(UpdateEvidenceCheckCurrent, fmt.Sprintf("already current at %s", shortCommit(current)))
}

func pluralCommitSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func shortCommit(sha string) string {
	if len(sha) <= 12 {
		return sha
	}
	return sha[:12]
}
```

- [x] **Step 5: Run GREEN tests**

Run:

```sh
go test ./internal/cli -run 'TestUpdateCommandCheck(FetchesRemote|ReportsAlready|FetchFailure)' -count=1
```

Expected: PASS.

## Task 3: Verify Command-Level JSON Still Works

**Files:**
- Modify only if tests fail: `cmd/gormes/update_command_test.go`, `cmd/gormes/update.go`

- [x] **Step 1: Run existing command JSON tests**

Run:

```sh
go test ./cmd/gormes -run 'TestUpdateCommand_JSON|TestUpdateCommandCheckMode|TestUpdateCommandUsesInjectedLifecycle' -count=1
```

Expected: PASS. The command uses injected lifecycle reports in these tests, so Slice 1 should not require command changes.

- [x] **Step 2: If `--check` copy test fails, update only the expected copy**

If a failure proves human output changed from "checked update readiness" to "checking remote update state", update the assertion to match the new copy. Do not weaken JSON parsing or exit-code assertions.

## Task 4: Focused Verification

**Files:**
- No planned edits.

- [x] **Step 1: Run focused update tests**

Run:

```sh
go test ./cmd/gormes ./internal/cli -run 'TestUpdate(Command|Lifecycle)|TestResolveBackupKeep|TestUpdateCommandCheck' -count=1
```

Expected: PASS.

- [x] **Step 2: Run repository metadata checks**

Run:

```sh
go run ./cmd/progress validate
git diff --check
```

Expected: both PASS.

- [x] **Step 3: Report remaining slices**

Report that Slice 1 is complete and that build/publish, active PATH refresh, gateway restart wiring, and update-log/SIGHUP protection remain for follow-up slices.
