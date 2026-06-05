package gormescli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// TestSystemCommands_JSONIncludeBuildProvenance proves
// `gormes system event/presence --json` and the snapshot read-back all
// emit a top-level `build` envelope so fleet automation pushing
// runtime presence/event state across machines can attribute each
// JSON document to the binary version that emitted it. Existing
// SystemEventResult/SystemPresenceResult/SystemEventsSnapshot fields
// remain addressable through struct embedding.
func TestSystemCommands_JSONIncludeBuildProvenance(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newSystemRootCommandForTest()
	stdout, stderr, err := executeRootCommandForTest(cmd, "system", "event", "gateway restart", "--mode", "now", "--json")
	if err != nil {
		t.Fatalf("system event --json: %v\nstderr=%s", err, stderr)
	}
	var eventGot struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		OK bool `json:"ok"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &eventGot); jsonErr != nil {
		t.Fatalf("event JSON decode: %v\nstdout=%s", jsonErr, stdout)
	}
	if eventGot.Build.Version != Version {
		t.Errorf("event build.version = %q, want %q", eventGot.Build.Version, Version)
	}
	if !eventGot.OK {
		t.Errorf("event ok = false, want true (still addressable)")
	}

	cmd2 := newSystemRootCommandForTest()
	stdout, stderr, err = executeRootCommandForTest(cmd2, "system", "presence", "--component", "gateway", "--status", "running", "--reason", "test", "--json")
	if err != nil {
		t.Fatalf("system presence --json: %v\nstderr=%s", err, stderr)
	}
	var presGot struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		OK bool `json:"ok"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &presGot); jsonErr != nil {
		t.Fatalf("presence JSON decode: %v\nstdout=%s", jsonErr, stdout)
	}
	if presGot.Build.Version != Version {
		t.Errorf("presence build.version = %q, want %q", presGot.Build.Version, Version)
	}
}

func newSystemRootCommandForTest() *cobra.Command {
	return newRootCommandWithFactoriesForTest(map[string]func() *cobra.Command{
		"system": func() *cobra.Command { return NewSystemCommand(testBuildProvenance) },
		"status": func() *cobra.Command {
			return NewStatusCommand(StatusCommandOptions{
				BuildProvenance: testBuildProvenance,
				SystemSnapshot: func(ctx context.Context) (toolspkg.SystemEventsSnapshot, error) {
					return DefaultSystemEventsManager().Snapshot(ctx)
				},
			})
		},
	})
}

func writeSystemStatusProgressFixture(t testing.TB, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "progress.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write progress fixture: %v", err)
	}
	return path
}

func TestSystemEventCommandWritesJSONAuditAndStatus(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newSystemRootCommandForTest()
	stdout, stderr, err := executeRootCommandForTest(cmd, "system", "event", "gateway restart", "--mode", "now", "--json")
	if err != nil {
		t.Fatalf("system event: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var eventResult toolspkg.SystemEventResult
	if err := json.Unmarshal([]byte(stdout), &eventResult); err != nil {
		t.Fatalf("decode event JSON: %v\nstdout=%s", err, stdout)
	}
	if !eventResult.OK || eventResult.Event.Text != "gateway restart" || !eventResult.Heartbeat.Triggered {
		t.Fatalf("event result = %+v, want enqueued event with heartbeat trigger", eventResult)
	}
	if _, err := os.Stat(config.ToolAuditLogPath()); err != nil {
		t.Fatalf("audit log was not written at %s: %v", config.ToolAuditLogPath(), err)
	}

	progress := writeSystemStatusProgressFixture(t, `{
  "meta": {"version": "2.0", "links": {"github_readme": "", "landing_page": "", "docs_site": "", "source_code": ""}},
  "phases": {}
}`)
	statusCmd := newSystemRootCommandForTest()
	statusOut, statusErr, err := executeRootCommandForTest(statusCmd, "status", "--progress", progress)
	if err != nil {
		t.Fatalf("status: %v\nstdout=%s\nstderr=%s", err, statusOut, statusErr)
	}
	for _, want := range []string{
		"system: heartbeat=enabled",
		"queued_events=1",
		"presence=",
		"tools/audit.jsonl",
	} {
		if !strings.Contains(statusOut, want) {
			t.Fatalf("status output missing %q:\n%s", want, statusOut)
		}
	}
}

func TestSystemHeartbeatAndPresenceCommands(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newSystemRootCommandForTest()
	stdout, stderr, err := executeRootCommandForTest(cmd, "system", "heartbeat", "disable", "--json")
	if err != nil {
		t.Fatalf("heartbeat disable: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var disabled toolspkg.SystemEventResult
	if err := json.Unmarshal([]byte(stdout), &disabled); err != nil {
		t.Fatalf("decode heartbeat disable: %v\nstdout=%s", err, stdout)
	}
	if !disabled.OK || disabled.Heartbeat.Enabled {
		t.Fatalf("disabled heartbeat = %+v, want ok disabled", disabled)
	}

	cmd = newSystemRootCommandForTest()
	stdout, stderr, err = executeRootCommandForTest(cmd, "system", "heartbeat", "enable", "--json")
	if err != nil {
		t.Fatalf("heartbeat enable: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var enabled toolspkg.SystemEventResult
	if err := json.Unmarshal([]byte(stdout), &enabled); err != nil {
		t.Fatalf("decode heartbeat enable: %v\nstdout=%s", err, stdout)
	}
	if !enabled.OK || !enabled.Heartbeat.Enabled {
		t.Fatalf("enabled heartbeat = %+v, want ok enabled", enabled)
	}

	cmd = newSystemRootCommandForTest()
	stdout, stderr, err = executeRootCommandForTest(cmd, "system", "presence", "--component", "gateway", "--status", "running", "--reason", "test", "--json")
	if err != nil {
		t.Fatalf("presence: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	var snapshot toolspkg.SystemEventsSnapshot
	if err := json.Unmarshal([]byte(stdout), &snapshot); err != nil {
		t.Fatalf("decode presence snapshot: %v\nstdout=%s", err, stdout)
	}
	if len(snapshot.Presence) == 0 || snapshot.Presence[0].Component != "gateway" || snapshot.Presence[0].Status != "running" {
		t.Fatalf("presence snapshot = %+v, want gateway running", snapshot.Presence)
	}
}
