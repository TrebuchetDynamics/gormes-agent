package delegate

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestParseTasksAcceptsJSONStringAndArray(t *testing.T) {
	fromString, err := ParseTasks(json.RawMessage(`"[{\"goal\":\"Research topic A\"},{\"goal\":\"Research topic B\",\"context\":\"channels only\",\"toolsets\":\"echo\"}]"`))
	if err != nil {
		t.Fatalf("ParseTasks string: %v", err)
	}
	if len(fromString) != 2 || fromString[0].Goal != "Research topic A" || fromString[1].Toolsets != "echo" {
		t.Fatalf("fromString = %#v", fromString)
	}

	fromArray, err := ParseTasks(json.RawMessage(`[{"goal":"alpha"},{"goal":"bravo","max_iterations":3}]`))
	if err != nil {
		t.Fatalf("ParseTasks array: %v", err)
	}
	if len(fromArray) != 2 || fromArray[1].MaxIterations != 3 {
		t.Fatalf("fromArray = %#v", fromArray)
	}
}

func TestParseTasksKeepsDelegateErrorGuidance(t *testing.T) {
	_, err := ParseTasks(json.RawMessage(`["not a task object"]`))
	if err == nil || !strings.Contains(err.Error(), "Task 0 must be an object") {
		t.Fatalf("err = %v, want object guidance", err)
	}

	_, err = ParseTasks(json.RawMessage(`"[{\"goal\":\"bad}"`))
	if err == nil || !strings.Contains(err.Error(), "could not be parsed as JSON") {
		t.Fatalf("err = %v, want JSON parse guidance", err)
	}
}

func TestResultEnvelopeAndToolsetHelpers(t *testing.T) {
	if got := SplitToolsets("a,b , c"); len(got) != 3 || got[1] != "b" {
		t.Fatalf("SplitToolsets = %#v", got)
	}
	if got := FirstPositive(0, -1, 3, 4); got != 3 {
		t.Fatalf("FirstPositive = %d, want 3", got)
	}

	out := ResultEnvelope(&Result{
		ID:           "sa_1",
		Status:       "completed",
		Summary:      "done",
		ExitReason:   "scripted",
		Duration:     12 * time.Millisecond,
		Iterations:   2,
		ToolCalls:    []string{"echo"},
		HasToolCalls: true,
	})
	if out["duration_ms"] != int64(12) || out["tool_calls"] == nil {
		t.Fatalf("ResultEnvelope = %#v", out)
	}

	nilOut := ResultEnvelope(nil)
	if nilOut["status"] != "error" || nilOut["exit_reason"] != "nil_result" {
		t.Fatalf("nil ResultEnvelope = %#v", nilOut)
	}
}
