package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestSystemEventCommandWritesJSONAuditAndStatus(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "system", "event", "gateway restart", "--mode", "now", "--json")
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

	progress := writeRootStatusProgressFixture(t, `{
  "meta": {"version": "2.0", "links": {"github_readme": "", "landing_page": "", "docs_site": "", "source_code": ""}},
  "phases": {}
}`)
	statusCmd := newRootCommandWithRuntime(rootRuntime{})
	statusOut, statusErr, err := executeOneshotFlagCommand(statusCmd, "status", "--progress", progress)
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

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "system", "heartbeat", "disable", "--json")
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

	cmd = newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err = executeOneshotFlagCommand(cmd, "system", "heartbeat", "enable", "--json")
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

	cmd = newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err = executeOneshotFlagCommand(cmd, "system", "presence", "--component", "gateway", "--status", "running", "--reason", "test", "--json")
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
