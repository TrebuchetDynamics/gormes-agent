package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestRootStatusCommand_JSONOnMissingProgressEmitsUnavailable pins
// the surface-symmetry contract: the text form of `gormes status` on
// a missing progress.json gracefully renders
// `blockers: unavailable status=progress_unavailable reason="..."`,
// but the `--json` form errored out with a raw filesystem message
// instead of a structured unavailable report. Fleet automation
// running `gormes status --json` to inventory blockers across a
// freshly-imaged host can't consume "Error: open: no such file"
// — it needs the same parseable shape.
//
// Contract: `--json` on missing progress emits a parseable document
// with a `progress` block carrying `status: "progress_unavailable"`
// and the original reason, without raising a non-zero exit (the
// missing file is a degraded state, not a CLI failure).
func TestRootStatusCommand_JSONOnMissingProgressEmitsUnavailable(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeRootCommandForTest(cmd, "status", "--progress", "/tmp/this/does/not/exist/progress.json", "--json")
	if err != nil {
		t.Fatalf("status --progress=missing --json on missing file must succeed; got %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}

	var got struct {
		Progress struct {
			Status string `json:"status"`
			Reason string `json:"reason"`
		} `json:"progress"`
		Blockers []any `json:"blockers"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}

	if got.Progress.Status != "progress_unavailable" {
		t.Errorf("progress.status = %q, want %q", got.Progress.Status, "progress_unavailable")
	}
	if !strings.Contains(got.Progress.Reason, "no such file") && got.Progress.Reason == "" {
		t.Errorf("progress.reason should reference the underlying error; got empty")
	}
	// Blockers must be `[]` (not null/missing) so consumers can
	// iterate without nil-checks — same convention as the rest of
	// the --json arc.
	if got.Blockers == nil {
		t.Fatalf(`blockers must decode to []any{}, not nil; got %v\nstdout=%s`, got.Blockers, stdout)
	}
}
