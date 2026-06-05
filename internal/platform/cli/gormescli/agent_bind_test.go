package gormescli

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestAgentBind_ResolvesByPeer proves `gormes agent bind` persists a
// runtime binding from a (channel, peer, thread) tuple to a dynamic agent
// and that `gormes agent inspect --json` resolves the same tuple back to
// the bound agent_id. Build provenance leads — same convention as the
// rest of the --json arc.
func TestAgentBind_ResolvesByPeer(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newAgentRootCommandForTest()
	if _, stderr, err := executeRootCommandForTest(cmd,
		"agent", "spawn", "Research", "--persona", "literature",
	); err != nil {
		t.Fatalf("spawn: %v\nstderr=%s", err, stderr)
	}

	cmd = newAgentRootCommandForTest()
	if _, stderr, err := executeRootCommandForTest(cmd,
		"agent", "bind", "research",
		"--channel", "telegram",
		"--peer-kind", "group",
		"--peer-id", "-100123",
		"--thread-id", "7",
	); err != nil {
		t.Fatalf("bind: %v\nstderr=%s", err, stderr)
	}

	cmd = newAgentRootCommandForTest()
	stdout, stderr, err := executeRootCommandForTest(cmd,
		"agent", "inspect",
		"--channel", "telegram",
		"--peer-kind", "group",
		"--peer-id", "-100123",
		"--thread-id", "7",
		"--json",
	)
	if err != nil {
		t.Fatalf("inspect: %v\nstderr=%s", err, stderr)
	}

	var report struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Match struct {
			Channel  string `json:"channel"`
			PeerKind string `json:"peer_kind"`
			PeerID   string `json:"peer_id"`
			ThreadID string `json:"thread_id"`
		} `json:"match"`
		Bound   bool   `json:"bound"`
		AgentID string `json:"agent_id"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &report); jsonErr != nil {
		t.Fatalf("invalid JSON: %v\nstdout=%s", jsonErr, stdout)
	}
	if report.Build.Version != Version {
		t.Errorf("build.version = %q, want %q", report.Build.Version, Version)
	}
	if !report.Bound {
		t.Errorf("bound = false, want true")
	}
	if report.AgentID != "research" {
		t.Errorf("agent_id = %q, want %q", report.AgentID, "research")
	}
	if report.Match.Channel != "telegram" || report.Match.PeerID != "-100123" || report.Match.ThreadID != "7" {
		t.Errorf("match echoed wrong: %+v", report.Match)
	}
}

// TestAgentUnbind_RemovesMatch proves `gormes agent unbind` clears the
// runtime binding and that `gormes agent inspect` then exits non-zero
// with agent_not_bound evidence on stderr (or in the JSON report when
// --json is used).
func TestAgentUnbind_RemovesMatch(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	cmd := newAgentRootCommandForTest()
	if _, stderr, err := executeRootCommandForTest(cmd,
		"agent", "spawn", "Research",
	); err != nil {
		t.Fatalf("spawn: %v\nstderr=%s", err, stderr)
	}
	cmd = newAgentRootCommandForTest()
	if _, stderr, err := executeRootCommandForTest(cmd,
		"agent", "bind", "research",
		"--channel", "telegram",
		"--peer-kind", "group",
		"--peer-id", "-100123",
		"--thread-id", "7",
	); err != nil {
		t.Fatalf("bind: %v\nstderr=%s", err, stderr)
	}
	cmd = newAgentRootCommandForTest()
	if _, stderr, err := executeRootCommandForTest(cmd,
		"agent", "unbind",
		"--channel", "telegram",
		"--peer-kind", "group",
		"--peer-id", "-100123",
		"--thread-id", "7",
	); err != nil {
		t.Fatalf("unbind: %v\nstderr=%s", err, stderr)
	}

	cmd = newAgentRootCommandForTest()
	stdout, stderr, err := executeRootCommandForTest(cmd,
		"agent", "inspect",
		"--channel", "telegram",
		"--peer-kind", "group",
		"--peer-id", "-100123",
		"--thread-id", "7",
	)
	if err == nil {
		t.Fatalf("inspect after unbind succeeded; want non-zero exit\nstdout=%s stderr=%s", stdout, stderr)
	}
	combined := stdout + stderr + err.Error()
	if !strings.Contains(combined, "agent_not_bound") {
		t.Errorf("expected agent_not_bound evidence after unbind, got stdout=%q stderr=%q err=%v",
			stdout, stderr, err)
	}

	// Idempotent unbind: removing an already-removed binding is a no-op.
	cmd = newAgentRootCommandForTest()
	if _, stderr, err := executeRootCommandForTest(cmd,
		"agent", "unbind",
		"--channel", "telegram",
		"--peer-kind", "group",
		"--peer-id", "-100123",
		"--thread-id", "7",
	); err != nil {
		t.Errorf("idempotent unbind err = %v\nstderr=%s", err, stderr)
	}
}
