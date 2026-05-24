package gonchotools

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/goncho"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestHonchoProfileEmptyHint_DefaultEmptyCard(t *testing.T) {
	reg, cleanup := newHonchoProfileHintRegistry(t, goncho.Config{
		WorkspaceID:    "default",
		ObserverPeerID: "gormes",
		RecentMessages: 4,
	})
	defer cleanup()

	output := executeHonchoTool(t, reg, "honcho_profile", json.RawMessage(`{"peer":"telegram:6586915095"}`))
	payload := decodeProfileHintPayload(t, output)

	if payload.WorkspaceID != "default" {
		t.Fatalf("workspace_id = %q, want default in %s", payload.WorkspaceID, output)
	}
	if payload.Peer != "telegram:6586915095" {
		t.Fatalf("peer = %q, want requested peer in %s", payload.Peer, output)
	}
	if payload.Result != "No profile facts available yet." {
		t.Fatalf("result = %q, want empty-card explanation in %s", payload.Result, output)
	}
	if payload.Hint == nil {
		t.Fatalf("hint missing in %s", output)
	}
	if payload.Hint.Code != "peer_card_empty_warmup" && payload.Hint.Code != "peer_card_empty_unknown" {
		t.Fatalf("hint code = %q, want peer_card_empty_warmup or peer_card_empty_unknown in %s", payload.Hint.Code, output)
	}
	if !strings.Contains(strings.ToLower(payload.Hint.Message), "not an error") {
		t.Fatalf("hint message = %q, want non-error guidance", payload.Hint.Message)
	}
}

func TestHonchoProfileEmptyHint_PeerCardDisabled(t *testing.T) {
	reg, cleanup := newHonchoProfileHintRegistry(t, goncho.Config{
		Enabled:         true,
		WorkspaceID:     "disabled-workspace",
		ObserverPeerID:  "gormes",
		PeerCardEnabled: false,
	})
	defer cleanup()

	output := executeHonchoTool(t, reg, "honcho_profile", json.RawMessage(`{"peer":"user"}`))
	payload := decodeProfileHintPayload(t, output)

	if payload.Hint == nil {
		t.Fatalf("hint missing in %s", output)
	}
	if payload.Hint.Code != "peer_card_disabled" {
		t.Fatalf("hint code = %q, want peer_card_disabled in %s", payload.Hint.Code, output)
	}
	if !strings.Contains(strings.ToLower(payload.Hint.Message), "disabled") {
		t.Fatalf("hint message = %q, want disabled evidence", payload.Hint.Message)
	}
}

func TestHonchoProfileEmptyHint_AlternativeToolsMentioned(t *testing.T) {
	reg, cleanup := newHonchoProfileHintRegistry(t, goncho.Config{
		WorkspaceID:    "default",
		ObserverPeerID: "gormes",
	})
	defer cleanup()

	output := executeHonchoTool(t, reg, "honcho_profile", json.RawMessage(`{"peer":"user"}`))
	payload := decodeProfileHintPayload(t, output)

	if payload.Hint == nil {
		t.Fatalf("hint missing in %s", output)
	}
	if !slices.Contains(payload.Hint.Alternatives, "honcho_reasoning") {
		t.Fatalf("alternatives = %v, want honcho_reasoning", payload.Hint.Alternatives)
	}
	if !slices.Contains(payload.Hint.Alternatives, "honcho_search") {
		t.Fatalf("alternatives = %v, want honcho_search", payload.Hint.Alternatives)
	}
}

func TestHonchoProfileEmptyHint_PopulatedCardOmitsHint(t *testing.T) {
	reg, svc, cleanup := newTestHonchoRegistry(t)
	defer cleanup()

	if err := svc.SetProfile(context.Background(), "user", []string{"Uses exact evidence-first reports"}); err != nil {
		t.Fatal(err)
	}

	output := executeHonchoTool(t, reg, "honcho_profile", json.RawMessage(`{"peer":"user"}`))
	payload := decodeProfileHintPayload(t, output)

	if !slices.Equal(payload.Card, []string{"Uses exact evidence-first reports"}) {
		t.Fatalf("card = %#v, want populated card in %s", payload.Card, output)
	}
	if payload.Hint != nil {
		t.Fatalf("hint = %+v, want omitted for populated card in %s", payload.Hint, output)
	}
	if payload.Result != "" {
		t.Fatalf("result = %q, want omitted for populated card in %s", payload.Result, output)
	}
}

func TestHonchoProfileEmptyHint_SchemaMentionsHintIsNonError(t *testing.T) {
	tool := &HonchoProfileTool{}

	var schema struct {
		Description string `json:"description"`
	}
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("schema unmarshal: %v", err)
	}
	lower := strings.ToLower(schema.Description)
	for _, want := range []string{"empty", "hint", "not an error"} {
		if !strings.Contains(lower, want) {
			t.Fatalf("schema description = %q, want %q", schema.Description, want)
		}
	}
}

type profileHintPayload struct {
	WorkspaceID string `json:"workspace_id"`
	Peer        string `json:"peer"`
	Card        []string
	Result      string `json:"result"`
	Hint        *struct {
		Code         string   `json:"code"`
		Message      string   `json:"message"`
		Alternatives []string `json:"alternatives"`
	} `json:"hint"`
}

func decodeProfileHintPayload(t *testing.T, output json.RawMessage) profileHintPayload {
	t.Helper()

	var payload profileHintPayload
	if err := json.Unmarshal(output, &payload); err != nil {
		t.Fatalf("profile output should be JSON object, got %s: %v", output, err)
	}
	return payload
}

func newHonchoProfileHintRegistry(t *testing.T, cfg goncho.Config) (*tools.Registry, func()) {
	t.Helper()

	store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	reg := tools.NewRegistry()
	svc := goncho.NewService(store.DB(), cfg, nil)
	RegisterHonchoTools(reg, svc)
	return reg, func() {
		if err := store.Close(context.Background()); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}
}
