package gormescli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func newTestStatusCommand(t *testing.T) *cobra.Command {
	t.Helper()
	auditPath := filepath.Join(t.TempDir(), "audit.log")
	return NewStatusCommand(StatusCommandOptions{
		BuildProvenance: func() BuildProvenance {
			return BuildProvenance{Version: "test-version", GitCommit: "test-sha"}
		},
		SystemSnapshot: func(context.Context) (toolspkg.SystemEventsSnapshot, error) {
			return toolspkg.SystemEventsSnapshot{}, nil
		},
		AuditPath: func() string { return auditPath },
	})
}

func executeStatusCommandForTest(cmd *cobra.Command, args ...string) (string, string, error) {
	return executeCobraCommandForTest(cmd, cobraCommandExecutionOptions{}, args...)
}

func TestStatusCommandShowsProgressBlockers(t *testing.T) {
	path := writeStatusProgressFixture(t, `{
  "meta": {"version": "2.0", "links": {"github_readme": "", "landing_page": "", "docs_site": "", "source_code": ""}},
  "phases": {
    "5": {
      "name": "Phase 5",
      "deliverable": "tools",
      "subphases": {
        "5.N": {
          "name": "operator tools",
          "items": [
            {
              "name": "blocked row",
              "status": "planned",
              "blocker": {
                "type": "infra",
                "status": "blocker_active",
                "blocker": "gateway lock",
                "evidence": "sessions.db locked",
                "unblocks_when": "lock exits",
                "owner": "operator",
                "pivot": "run next P0 row",
                "next_check": "2026-05-01T12:30:00-06:00"
              }
            }
          ]
        }
      }
    }
  }
}`)

	cmd := newTestStatusCommand(t)
	stdout, stderr, err := executeStatusCommandForTest(cmd, "--progress", path)
	if err != nil {
		t.Fatalf("gormes status failed: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Gormes Status",
		"blockers: 1 active",
		"5/5.N blocked row type=infra owner=operator status=blocker_active",
		"evidence: sessions.db locked",
		"workaround/pivot: run next P0 row",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("gormes status output missing %q:\n%s", want, stdout)
		}
	}
}

// TestStatusCommand_JSONIncludesBuildProvenance proves
// `gormes status --json` carries the running binary's build SHA and
// version. Same contract as update/doctor — captured status
// snapshots stay attributable to a specific binary.
func TestStatusCommand_JSONIncludesBuildProvenance(t *testing.T) {
	path := writeStatusProgressFixture(t, `{
  "meta": {"version": "2.0", "links": {"github_readme": "", "landing_page": "", "docs_site": "", "source_code": ""}},
  "phases": {}
}`)
	cmd := newTestStatusCommand(t)
	stdout, _, err := executeStatusCommandForTest(cmd, "--progress", path, "--json")
	if err != nil {
		t.Fatalf("status --json: %v", err)
	}
	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Build.Version != "test-version" {
		t.Fatalf("got.build.version = %q, want %q", got.Build.Version, "test-version")
	}
	if got.Build.GitCommit != "test-sha" {
		t.Fatalf("got.build.git_commit = %q, want %q", got.Build.GitCommit, "test-sha")
	}
}

// TestStatusCommand_JSONEmitsStructuredBlockers proves
// `gormes status --json` returns a parseable {"blockers": [...]}
// document with phase, subphase, row, and the BlockerRecord fields.
// CI/cron consumers monitoring active blockers can ingest this without
// scraping the human-readable rows.
func TestStatusCommand_JSONEmitsStructuredBlockers(t *testing.T) {
	path := writeStatusProgressFixture(t, `{
  "meta": {"version": "2.0", "links": {"github_readme": "", "landing_page": "", "docs_site": "", "source_code": ""}},
  "phases": {
    "5": {
      "name": "Phase 5",
      "deliverable": "tools",
      "subphases": {
        "5.N": {
          "name": "operator tools",
          "items": [
            {
              "name": "blocked row",
              "status": "planned",
              "blocker": {
                "type": "infra",
                "status": "blocker_active",
                "blocker": "gateway lock",
                "evidence": "sessions.db locked",
                "unblocks_when": "lock exits",
                "owner": "operator",
                "pivot": "run next P0 row",
                "next_check": "2026-05-08T12:30:00-06:00"
              }
            }
          ]
        }
      }
    }
  }
}`)

	cmd := newTestStatusCommand(t)
	stdout, stderr, err := executeStatusCommandForTest(cmd, "--progress", path, "--json")
	if err != nil {
		t.Fatalf("gormes status --json: %v stderr=%s stdout=%s", err, stderr, stdout)
	}

	var got struct {
		Blockers []struct {
			Phase    string `json:"phase"`
			Subphase string `json:"subphase"`
			Row      string `json:"row"`
			Title    string `json:"title"`
			Type     string `json:"type"`
			Status   string `json:"status"`
			Blocker  string `json:"blocker"`
			Evidence string `json:"evidence"`
			Owner    string `json:"owner"`
		} `json:"blockers"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if len(got.Blockers) != 1 {
		t.Fatalf("got %d blockers, want 1", len(got.Blockers))
	}
	b := got.Blockers[0]
	if b.Phase != "5" || b.Subphase != "5.N" || b.Row != "blocked row" {
		t.Fatalf("unexpected coordinates: phase=%q subphase=%q row=%q", b.Phase, b.Subphase, b.Row)
	}
	if b.Type != "infra" || b.Status != "blocker_active" || b.Owner != "operator" {
		t.Fatalf("unexpected record: type=%q status=%q owner=%q", b.Type, b.Status, b.Owner)
	}
	// JSON mode must not also emit the human header.
	if strings.Contains(stdout, "Gormes Status") {
		t.Fatalf("--json must not emit the human header; got:\n%s", stdout)
	}
}

// TestStatusCommand_JSONNoBlockersEmitsEmptyArray proves the JSON
// surface stays parseable when no blockers exist — consumers see
// `{"blockers": []}`, not a free-form "blockers: none" message.
func TestStatusCommand_JSONNoBlockersEmitsEmptyArray(t *testing.T) {
	path := writeStatusProgressFixture(t, `{
  "meta": {"version": "2.0", "links": {"github_readme": "", "landing_page": "", "docs_site": "", "source_code": ""}},
  "phases": {}
}`)
	cmd := newTestStatusCommand(t)
	stdout, _, err := executeStatusCommandForTest(cmd, "--progress", path, "--json")
	if err != nil {
		t.Fatalf("gormes status --json: %v", err)
	}
	var got struct {
		Blockers []any `json:"blockers"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("empty stdout must be valid JSON; got %q\nerr=%v", stdout, jsonErr)
	}
	if got.Blockers == nil {
		t.Fatalf("blockers must be `[]`, not omitted/null; got %q", stdout)
	}
	if len(got.Blockers) != 0 {
		t.Fatalf("got %d blockers, want 0", len(got.Blockers))
	}
}

// TestStatusCommand_JSONIncludesSystemSnapshot pins the wire
// shape symmetry contract: the human surface of `gormes status`
// already prints a `system: heartbeat=… queued_events=… presence=…
// audit=…` line via RenderSystemStatusLine, but the `--json` surface
// must not drop it. Fleet automation that ingests `--json` to alert
// on system-event backlog or audit log location should not scrape the
// human-readable line.
func TestStatusCommand_JSONIncludesSystemSnapshot(t *testing.T) {
	path := writeStatusProgressFixture(t, `{
  "meta": {"version": "2.0", "links": {"github_readme": "", "landing_page": "", "docs_site": "", "source_code": ""}},
  "phases": {}
}`)

	cmd := newTestStatusCommand(t)
	stdout, stderr, err := executeStatusCommandForTest(cmd, "--progress", path, "--json")
	if err != nil {
		t.Fatalf("status --json: %v\nstderr=%s", err, stderr)
	}

	var got struct {
		System struct {
			Heartbeat struct {
				Enabled bool `json:"enabled"`
			} `json:"heartbeat"`
			Events   []any `json:"events"`
			Presence []any `json:"presence"`
		} `json:"system"`
		AuditPath string `json:"audit_path"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%s", jsonErr, stdout)
	}

	// The text surface always prints `audit=<path>`, even if the
	// path doesn't exist yet — so JSON must populate it too.
	if got.AuditPath == "" {
		t.Fatalf("`audit_path` must be populated to match the text surface; got empty\nstdout=%s", stdout)
	}
	if !strings.Contains(got.AuditPath, "audit") {
		t.Errorf("`audit_path` looks suspicious — should contain `audit`; got %q", got.AuditPath)
	}
	// Events/Presence may be empty arrays on a fresh install, but
	// they must be present (not omitted) so consumers can iterate
	// without nil-checks.
	if got.System.Events == nil {
		t.Errorf("`system.events` must be `[]`, not omitted/null on fresh install; stdout=%s", stdout)
	}
	if got.System.Presence == nil {
		t.Errorf("`system.presence` must be `[]`, not omitted/null on fresh install; stdout=%s", stdout)
	}
}

// TestStatusCommand_JSONOnMissingProgressEmitsUnavailable pins
// the surface-symmetry contract: the text form of `gormes status` on
// a missing progress.json gracefully renders
// `blockers: unavailable status=progress_unavailable reason="..."`,
// but the `--json` form must also produce structured unavailable
// status instead of a raw filesystem error.
func TestStatusCommand_JSONOnMissingProgressEmitsUnavailable(t *testing.T) {
	cmd := newTestStatusCommand(t)
	stdout, stderr, err := executeStatusCommandForTest(cmd, "--progress", "/tmp/this/does/not/exist/progress.json", "--json")
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
	if got.Progress.Reason == "" || !strings.Contains(got.Progress.Reason, "no such file") {
		t.Errorf("progress.reason should reference the underlying error; got %q", got.Progress.Reason)
	}
	// Blockers must be `[]` (not null/missing) so consumers can
	// iterate without nil-checks — same convention as the rest of
	// the --json arc.
	if got.Blockers == nil {
		t.Fatalf(`blockers must decode to []any{}, not nil; got %v\nstdout=%s`, got.Blockers, stdout)
	}
}

func writeStatusProgressFixture(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "progress.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write progress fixture: %v", err)
	}
	return path
}
