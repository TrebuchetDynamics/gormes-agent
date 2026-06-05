package cronjob_test

import (
	"encoding/json"
	"testing"

	toolcron "github.com/TrebuchetDynamics/gormes-agent/internal/tools/cronjob"
)

func TestCronjobToolNoAgent_CreateRequiresScriptButNotPrompt(t *testing.T) {
	store, done := newCronjobToolTestStore(t)
	defer done()

	tool := toolcron.NewCronjobTool(toolcron.CronjobToolConfig{
		Store:       store,
		ScriptsRoot: t.TempDir(),
		Now:         fixedCronjobToolNow,
	})

	assertCronjobToolError(t, tool, map[string]any{
		"action":   "create",
		"schedule": "every 5m",
		"no_agent": true,
	}, "no_agent=True requires a script")

	created := execCronjobTool[cronjobCreateResult](t, tool, map[string]any{
		"action":   "create",
		"name":     "watchdog",
		"schedule": "every 5m",
		"script":   "watchdog.sh",
		"no_agent": true,
	})
	if !created.Success {
		t.Fatalf("create success = false, error = %q", created.Error)
	}
	stored, err := store.Get(created.JobID)
	if err != nil {
		t.Fatalf("store.Get(created) = %v", err)
	}
	if !stored.NoAgent || stored.Script != "watchdog.sh" || stored.Prompt != "" {
		t.Fatalf("stored no-agent job = %+v, want no_agent script-only job", stored)
	}

	var listed struct {
		Success bool `json:"success"`
		Jobs    []struct {
			NoAgent bool `json:"no_agent"`
		} `json:"jobs"`
	}
	rawList := execCronjobToolRaw(t, tool, map[string]any{"action": "list"})
	if err := json.Unmarshal(rawList, &listed); err != nil {
		t.Fatalf("unmarshal list output %s: %v", rawList, err)
	}
	if len(listed.Jobs) != 1 || !listed.Jobs[0].NoAgent {
		t.Fatalf("list jobs = %+v, want no_agent summary", listed.Jobs)
	}
}

func TestCronjobToolNoAgent_UpdateRequiresPromptWhenReEnablingAgent(t *testing.T) {
	store, done := newCronjobToolTestStore(t)
	defer done()

	tool := toolcron.NewCronjobTool(toolcron.CronjobToolConfig{
		Store:       store,
		ScriptsRoot: t.TempDir(),
		Now:         fixedCronjobToolNow,
	})

	created := execCronjobTool[cronjobCreateResult](t, tool, map[string]any{
		"action":   "create",
		"name":     "script-only",
		"schedule": "every 30m",
		"script":   "watchdog.sh",
		"no_agent": true,
	})

	assertCronjobToolError(t, tool, map[string]any{
		"action":   "update",
		"job_id":   created.JobID,
		"no_agent": false,
	}, "prompt is required")
}

func TestCronjobToolNoAgent_UpdateTogglesWithScriptGuard(t *testing.T) {
	store, done := newCronjobToolTestStore(t)
	defer done()

	tool := toolcron.NewCronjobTool(toolcron.CronjobToolConfig{
		Store:       store,
		ScriptsRoot: t.TempDir(),
		Now:         fixedCronjobToolNow,
	})

	agentJob := execCronjobTool[cronjobCreateResult](t, tool, map[string]any{
		"action":   "create",
		"name":     "agent",
		"schedule": "every 30m",
		"prompt":   "Summarize changes.",
	})
	assertCronjobToolError(t, tool, map[string]any{
		"action":   "update",
		"job_id":   agentJob.JobID,
		"no_agent": true,
	}, "without a script")

	var updated struct {
		Success bool `json:"success"`
		Job     struct {
			NoAgent bool `json:"no_agent"`
		} `json:"job"`
	}
	rawUpdated := execCronjobToolRaw(t, tool, map[string]any{
		"action":   "update",
		"job_id":   agentJob.JobID,
		"script":   "watchdog.sh",
		"no_agent": true,
	})
	if err := json.Unmarshal(rawUpdated, &updated); err != nil {
		t.Fatalf("unmarshal update output %s: %v", rawUpdated, err)
	}
	if !updated.Success || !updated.Job.NoAgent {
		t.Fatalf("update = %+v, want no_agent true", updated)
	}
	stored, err := store.Get(agentJob.JobID)
	if err != nil {
		t.Fatalf("store.Get(updated) = %v", err)
	}
	if !stored.NoAgent || stored.Script != "watchdog.sh" || stored.Prompt != "" {
		t.Fatalf("stored updated job = %+v, want no_agent script-only job with cleared prompt", stored)
	}

	var off struct {
		Success bool `json:"success"`
		Job     struct {
			NoAgent bool `json:"no_agent"`
		} `json:"job"`
	}
	rawOff := execCronjobToolRaw(t, tool, map[string]any{
		"action":   "update",
		"job_id":   agentJob.JobID,
		"no_agent": false,
		"prompt":   "Run through the agent again.",
	})
	if err := json.Unmarshal(rawOff, &off); err != nil {
		t.Fatalf("unmarshal update-off output %s: %v", rawOff, err)
	}
	if !off.Success || off.Job.NoAgent {
		t.Fatalf("update off = %+v, want no_agent false", off)
	}
}
