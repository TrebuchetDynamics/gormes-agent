package gonchotools

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/goncho/service"
)

func TestGonchoProofMatrix_HonchoToolsExerciseLocalGonchoContracts(t *testing.T) {
	reg, _, cleanup := newTestHonchoRegistry(t)
	defer cleanup()

	for _, name := range []string{"honcho_profile", "honcho_search", "honcho_context", "honcho_reasoning", "honcho_conclude"} {
		if _, ok := reg.Get(name); !ok {
			t.Fatalf("%s not registered", name)
		}
	}
	if _, ok := reg.Get("honcho_chat"); ok {
		t.Fatal("honcho_chat must remain unregistered as a tool; use Service.Chat for dialectic hosts")
	}

	profileOutput := executeHonchoTool(t, reg, "honcho_profile", json.RawMessage(`{
		"peer":"user-juan",
		"card":["Prefers deterministic Goncho proof fixtures"]
	}`))
	var profile goncho.ProfileResult
	if err := json.Unmarshal(profileOutput, &profile); err != nil {
		t.Fatalf("profile JSON: %v\n%s", err, profileOutput)
	}
	if !slices.Equal(profile.Card, []string{"Prefers deterministic Goncho proof fixtures"}) {
		t.Fatalf("profile = %+v", profile)
	}

	concludeOutput := executeHonchoTool(t, reg, "honcho_conclude", json.RawMessage(`{
		"peer":"user-juan",
		"conclusion":"Goncho Honcho tools prove local memory through codename violet.",
		"session_key":"sess-proof"
	}`))
	var conclusion goncho.ConcludeResult
	if err := json.Unmarshal(concludeOutput, &conclusion); err != nil {
		t.Fatalf("conclusion JSON: %v\n%s", err, concludeOutput)
	}
	if conclusion.WorkspaceID != "default" || conclusion.Status == "" || conclusion.ID == 0 {
		t.Fatalf("conclusion = %+v", conclusion)
	}

	searchOutput := executeHonchoTool(t, reg, "honcho_search", json.RawMessage(`{
		"peer":"user-juan",
		"query":"codename violet",
		"session_key":"sess-proof",
		"max_tokens":200
	}`))
	var search goncho.SearchResultSet
	if err := json.Unmarshal(searchOutput, &search); err != nil {
		t.Fatalf("search JSON: %v\n%s", err, searchOutput)
	}
	if len(search.Results) != 1 || !strings.Contains(search.Results[0].Content, "codename violet") {
		t.Fatalf("search = %+v", search)
	}

	contextOutput := executeHonchoTool(t, reg, "honcho_context", json.RawMessage(`{
		"peer":"user-juan",
		"query":"codename violet",
		"session_key":"sess-proof",
		"max_tokens":400
	}`))
	var contextResult goncho.ContextResult
	if err := json.Unmarshal(contextOutput, &contextResult); err != nil {
		t.Fatalf("context JSON: %v\n%s", err, contextOutput)
	}
	if len(contextResult.PeerCard) != 1 || len(contextResult.Conclusions) != 1 || contextResult.Representation == "" {
		t.Fatalf("context = %+v", contextResult)
	}

	reasoningOutput := executeHonchoTool(t, reg, "honcho_reasoning", json.RawMessage(`{
		"peer":"user-juan",
		"query":"How should Goncho be tested?",
		"session_key":"sess-proof",
		"reasoning_level":"low"
	}`))
	var reasoning struct {
		WorkspaceID string `json:"workspace_id"`
		Peer        string `json:"peer"`
		Answer      string `json:"answer"`
		Evidence    string `json:"evidence"`
	}
	if err := json.Unmarshal(reasoningOutput, &reasoning); err != nil {
		t.Fatalf("reasoning JSON: %v\n%s", err, reasoningOutput)
	}
	if reasoning.WorkspaceID != "default" || reasoning.Peer != "user-juan" || !strings.Contains(reasoning.Answer, "codename violet") {
		t.Fatalf("reasoning = %+v", reasoning)
	}
	if reasoning.Evidence != "reasoning_llm_unavailable" {
		t.Fatalf("reasoning evidence = %q, want deterministic fallback evidence", reasoning.Evidence)
	}
}
