package gormescli

import (
	"encoding/json"
	"testing"
)

// TestAgentInspect_JSONReportsUnboundCleanly proves that `gormes agent
// inspect --json` on an unbound peer exits 0 and emits a parseable
// report with bound=false (no agent_id). Fleet automation parses this
// shape to branch on "is this thread routed to a runtime agent yet?"
// without paying attention to exit codes.
func TestAgentInspect_JSONReportsUnboundCleanly(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newAgentRootCommandForTest()

	stdout, stderr, err := executeRootCommandForTest(cmd,
		"agent", "inspect",
		"--channel", "telegram",
		"--peer-kind", "group",
		"--peer-id", "-100123",
		"--thread-id", "7",
		"--json",
	)
	if err != nil {
		t.Fatalf("inspect --json on unbound peer: %v\nstderr=%s", err, stderr)
	}

	var report struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Match struct {
			Channel  string `json:"channel"`
			PeerID   string `json:"peer_id"`
			ThreadID string `json:"thread_id"`
		} `json:"match"`
		Bound   *bool  `json:"bound"`
		AgentID string `json:"agent_id"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &report); jsonErr != nil {
		t.Fatalf("invalid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if report.Build.Version != Version {
		t.Errorf("build.version = %q, want %q", report.Build.Version, Version)
	}
	if report.Bound == nil {
		t.Fatalf("bound field missing; want explicit false")
	}
	if *report.Bound {
		t.Errorf("bound = true, want false for unbound peer")
	}
	if report.AgentID != "" {
		t.Errorf("agent_id = %q, want empty for unbound peer", report.AgentID)
	}
	if report.Match.Channel != "telegram" || report.Match.PeerID != "-100123" || report.Match.ThreadID != "7" {
		t.Errorf("match echoed wrong: %+v", report.Match)
	}
}
