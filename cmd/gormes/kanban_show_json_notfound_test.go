package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestKanbanShow_NotFoundJSONEmitsStructuredDocument pins the
// regression observed during a fresh-install probe sweep:
// `gormes kanban show <missing> --json` emitted nothing on stdout
// — only `Error: kanban task "missing" not found` on stderr via
// cobra's standard rendering. Fleet automation scraping
// `gormes kanban show ID --json` to inventory task state across
// machines couldn't distinguish "task missing" from "command
// crashed" — both yielded empty stdout + non-zero exit.
//
// Contract: when --json is set and the task isn't found, stdout
// MUST carry a parseable `{build, action: "not_found", id, error}`
// document. The non-zero exit code stays (it's still a failed
// lookup), but the structured outcome is now ingestible.
//
// Same convention as `session delete --json` (which already emits
// `action: "not_found"`) and the mcp login JSON path. Provides a
// uniform surface across every "look up X" --json command.
func TestKanbanShow_NotFoundJSONEmitsStructuredDocument(t *testing.T) {
	for _, verb := range []string{"show", "complete", "claim"} {
		verb := verb
		t.Run(verb, func(t *testing.T) {
			freshInstallE2EHome(t)
			// The kanban store opens lazily; init it so the
			// not-found path is reached (rather than "DB doesn't
			// exist").
			cmd := newRootCommandWithRuntime(rootRuntime{})
			if _, _, err := executeRootCommandForTest(cmd, "kanban", "init"); err != nil {
				t.Fatalf("kanban init: %v", err)
			}

			cmd = newRootCommandWithRuntime(rootRuntime{})
			stdout, stderr, err := executeRootCommandForTest(cmd, "kanban", verb, "definitely-missing-task-id", "--json")
			// Exit non-zero is correct — the lookup failed.
			if err == nil {
				t.Fatalf("kanban %s <missing> --json must error; stdout=%q stderr=%q", verb, stdout, stderr)
			}

			if strings.TrimSpace(stdout) == "" {
				t.Fatalf("kanban %s --json must emit a JSON document on stdout even on not-found; got empty stdout. stderr=%s", verb, stderr)
			}

			var got struct {
				Build struct {
					Version string `json:"version"`
				} `json:"build"`
				Action string `json:"action"`
				ID     string `json:"id"`
			}
			if jsonErr := json.Unmarshal([]byte(stdout), &got); jsonErr != nil {
				t.Fatalf("stdout must be parseable JSON: %v\nstdout=%s", jsonErr, stdout)
			}
			if got.Build.Version != Version {
				t.Errorf("build.version = %q, want %q", got.Build.Version, Version)
			}
			if got.Action != "not_found" {
				t.Errorf("action = %q, want %q", got.Action, "not_found")
			}
			if got.ID != "definitely-missing-task-id" {
				t.Errorf("id = %q, want the requested id", got.ID)
			}
		})
	}
}
