package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestRootStatusCommand_JSONIncludesSystemSnapshot pins the wire
// shape symmetry contract: the human surface of `gormes status`
// already prints a `system: heartbeat=… queued_events=… presence=…
// audit=…` line via renderSystemStatusLine, but the `--json` surface
// silently drops it. Fleet automation that ingests `--json` to alert
// on system-event backlog or audit log location is then forced to
// scrape the human-readable line — which is the exact failure mode
// `--json` was added to prevent.
//
// Contract: `--json` must include a `system` block carrying the same
// snapshot the text form prints, plus the `audit_path` field the
// text line surfaces.
func TestRootStatusCommand_JSONIncludesSystemSnapshot(t *testing.T) {
	path := writeRootStatusProgressFixture(t, `{
  "meta": {"version": "2.0", "links": {"github_readme": "", "landing_page": "", "docs_site": "", "source_code": ""}},
  "phases": {}
}`)
	t.Setenv("GORMES_HOME", t.TempDir())

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeRootCommandForTest(cmd, "status", "--progress", path, "--json")
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
