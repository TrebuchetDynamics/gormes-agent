package main

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"testing"
)

func TestGonchoProofMatrix_CommandSurfacesDoctorDiagnosticsAndReplay(t *testing.T) {
	seedGonchoDoctorZeroStateDB(t)

	doctorOut, doctorErrOut, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}),
		"goncho", "doctor", "--json", "--peer=user-juan", "--session=sess-proof",
	)
	if err != nil {
		t.Fatalf("goncho doctor --json: %v\nstdout=%s\nstderr=%s", err, doctorOut, doctorErrOut)
	}
	if doctorErrOut != "" {
		t.Fatalf("doctor stderr = %q, want empty", doctorErrOut)
	}
	var doctor struct {
		Service          string `json:"service"`
		Status           string `json:"status"`
		ToolRegistration struct {
			Registered []string `json:"registered"`
		} `json:"tool_registration"`
		ContextDryRun struct {
			Peer       string `json:"peer"`
			SessionKey string `json:"session_key"`
		} `json:"context_dry_run"`
		DegradedModes []struct {
			Capability string `json:"capability"`
			Severity   string `json:"severity"`
		} `json:"degraded_modes"`
	}
	if err := json.Unmarshal([]byte(doctorOut), &doctor); err != nil {
		t.Fatalf("doctor JSON: %v\n%s", err, doctorOut)
	}
	if doctor.Service != "goncho" || doctor.Status != "degraded" {
		t.Fatalf("doctor header = %+v", doctor)
	}
	for _, want := range []string{"honcho_profile", "honcho_search", "honcho_context", "honcho_reasoning", "honcho_conclude"} {
		if !slices.Contains(doctor.ToolRegistration.Registered, want) {
			t.Fatalf("doctor registered tools = %+v, missing %s", doctor.ToolRegistration.Registered, want)
		}
	}
	if doctor.ContextDryRun.Peer != "user-juan" || doctor.ContextDryRun.SessionKey != "sess-proof" {
		t.Fatalf("doctor context dry-run = %+v", doctor.ContextDryRun)
	}
	if len(doctor.DegradedModes) == 0 {
		t.Fatal("doctor degraded modes empty; zero-state Goncho must expose operator-visible degraded evidence")
	}

	tracePath := filepath.Join("..", "..", "internal", "goncho", "testdata", "recall_trace", "stable_trace.golden.json")
	diagnosticsOut, diagnosticsErrOut, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}),
		"goncho", "recall-diagnostics", "--trace", tracePath, "--json",
	)
	if err != nil {
		t.Fatalf("goncho recall-diagnostics --json: %v\nstdout=%s\nstderr=%s", err, diagnosticsOut, diagnosticsErrOut)
	}
	if diagnosticsErrOut != "" {
		t.Fatalf("diagnostics stderr = %q, want empty", diagnosticsErrOut)
	}
	var diagnostics struct {
		Service             string `json:"service"`
		Status              string `json:"status"`
		TraceID             string `json:"trace_id"`
		ProjectionInvariant string `json:"projection_invariant"`
		Selected            []struct {
			MemoryID string `json:"memory_id"`
		} `json:"selected"`
		Rejected []struct {
			MemoryID string `json:"memory_id"`
			Reason   string `json:"reason"`
		} `json:"rejected"`
	}
	if err := json.Unmarshal([]byte(diagnosticsOut), &diagnostics); err != nil {
		t.Fatalf("diagnostics JSON: %v\n%s", err, diagnosticsOut)
	}
	if diagnostics.Service != "goncho" || diagnostics.Status != "ok" || diagnostics.TraceID == "" || diagnostics.ProjectionInvariant != "no_projection_without_recall_trace" {
		t.Fatalf("diagnostics = %+v", diagnostics)
	}
	if len(diagnostics.Selected) != 2 || len(diagnostics.Rejected) != 1 || diagnostics.Rejected[0].Reason != "not_selected" {
		t.Fatalf("diagnostics selected/rejected = selected %+v rejected %+v", diagnostics.Selected, diagnostics.Rejected)
	}

	replayOut, replayErrOut, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}),
		"goncho", "recall-replay", "--trace", tracePath, "--json",
	)
	if err != nil {
		t.Fatalf("goncho recall-replay --json: %v\nstdout=%s\nstderr=%s", err, replayOut, replayErrOut)
	}
	if replayErrOut != "" {
		t.Fatalf("replay stderr = %q, want empty", replayErrOut)
	}
	var replay struct {
		Service             string `json:"service"`
		TraceID             string `json:"trace_id"`
		EventCount          int    `json:"event_count"`
		ProjectionInvariant string `json:"projection_invariant"`
		ReplayContract      string `json:"replay_contract"`
		Events              []struct {
			Index    int    `json:"index"`
			Kind     string `json:"kind"`
			MemoryID string `json:"memory_id"`
			Reason   string `json:"reason"`
		} `json:"events"`
	}
	if err := json.Unmarshal([]byte(replayOut), &replay); err != nil {
		t.Fatalf("replay JSON: %v\n%s", err, replayOut)
	}
	if replay.Service != "goncho" || replay.TraceID != diagnostics.TraceID || replay.ProjectionInvariant != "no_projection_without_recall_trace" || replay.ReplayContract != "deterministic_replay_from_recall_trace" {
		t.Fatalf("replay = %+v", replay)
	}
	if replay.EventCount != len(replay.Events) || replay.EventCount == 0 {
		t.Fatalf("replay event count = %d len=%d", replay.EventCount, len(replay.Events))
	}
	for i, event := range replay.Events {
		if event.Index != i+1 {
			t.Fatalf("replay event[%d] index=%d, want %d", i, event.Index, i+1)
		}
	}
	if replay.Events[0].Kind != "recall_query" || replay.Events[len(replay.Events)-1].Kind != "projection_ready" {
		t.Fatalf("replay events = %+v, want query first and projection_ready last", replay.Events)
	}
	if !replayHasRejectedMemory(replay.Events, "mem-rate", "not_selected") {
		t.Fatalf("replay events = %+v, missing rejected mem-rate not_selected", replay.Events)
	}
}

func replayHasRejectedMemory(events []struct {
	Index    int    `json:"index"`
	Kind     string `json:"kind"`
	MemoryID string `json:"memory_id"`
	Reason   string `json:"reason"`
}, memoryID, reason string) bool {
	for _, event := range events {
		if event.Kind == "rejected" && event.MemoryID == memoryID && event.Reason == reason {
			return true
		}
	}
	return false
}
