package main

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

// TestAgentSpawn_PersistsAndListsRecord proves `gormes agent spawn <name>`
// returns the normalized AgentID, persists the record in the dynamic
// registry under $GORMES_HOME, and that `gormes agent list --json` echoes
// the record with leading build provenance (same convention as the rest
// of the --json arc).
func TestAgentSpawn_PersistsAndListsRecord(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})

	stdout, stderr, err := executeOneshotFlagCommand(cmd,
		"agent", "spawn", "Research Bot",
		"--persona", "literature review",
	)
	if err != nil {
		t.Fatalf("agent spawn: %v\nstderr=%s", err, stderr)
	}
	if !strings.Contains(stdout, "research-bot") {
		t.Errorf("spawn stdout missing normalized id research-bot:\n%s", stdout)
	}

	cmd = newRootCommandWithRuntime(rootRuntime{})
	listStdout, listStderr, err := executeOneshotFlagCommand(cmd, "agent", "list", "--json")
	if err != nil {
		t.Fatalf("agent list --json: %v\nstderr=%s", err, listStderr)
	}
	var listReport struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Agents []struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Persona   string `json:"persona"`
			CreatedAt string `json:"created_at"`
		} `json:"agents"`
	}
	if jsonErr := json.Unmarshal([]byte(listStdout), &listReport); jsonErr != nil {
		t.Fatalf("agent list --json must be valid JSON: %v\nstdout=%s", jsonErr, listStdout)
	}
	if listReport.Build.Version != Version {
		t.Errorf("build.version = %q, want %q", listReport.Build.Version, Version)
	}
	if len(listReport.Agents) != 1 {
		t.Fatalf("agents len = %d, want 1\nstdout=%s", len(listReport.Agents), listStdout)
	}
	a := listReport.Agents[0]
	if a.ID != "research-bot" || a.Name != "Research Bot" || a.Persona != "literature review" {
		t.Errorf("agent[0] = %+v, want id=research-bot name='Research Bot' persona='literature review'", a)
	}
	if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`).MatchString(a.CreatedAt) {
		t.Errorf("created_at = %q, want RFC3339-shaped timestamp", a.CreatedAt)
	}
}

// TestAgentSpawnRejectsInvalidID proves `gormes agent spawn` returns a
// non-zero exit when the name does not normalize to a config-compatible
// AgentID (^[a-z][a-z0-9_-]{0,63}$) — operator typos must be reported
// instead of silently producing an unreachable agent.
func TestAgentSpawnRejectsInvalidID(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	cmd := newRootCommandWithRuntime(rootRuntime{})

	stdout, stderr, err := executeOneshotFlagCommand(cmd,
		"agent", "spawn", "!!!",
	)
	if err == nil {
		t.Fatalf("spawn with invalid name succeeded; stdout=%s stderr=%s", stdout, stderr)
	}
	combined := stdout + stderr + err.Error()
	if !strings.Contains(combined, "agent_id_invalid") {
		t.Errorf("expected agent_id_invalid evidence, got stdout=%q stderr=%q err=%v",
			stdout, stderr, err)
	}
}
