package navivox

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
)

func TestNavivoxMemoryOverviewRequiresAuthAndReturnsSafeCounts(t *testing.T) {
	memoryDB := filepath.Join(t.TempDir(), "private", "mineru", "memory.db")
	store, err := memory.OpenSqlite(memoryDB, 8, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(t.Context())

	if _, err := store.DB().Exec(`INSERT INTO turns(session_id, role, content, ts_unix, chat_id) VALUES
		('s-1', 'user', 'remember my keyboard layout', 1710000001, 'chat-1'),
		('s-1', 'assistant', 'noted', 1710000002, 'chat-1')`); err != nil {
		t.Fatalf("seed turns: %v", err)
	}
	if _, err := store.DB().Exec(`INSERT INTO entities(name, type, updated_at) VALUES
		('Mineru', 'PERSON', 1710000003),
		('Goncho', 'PROJECT', 1710000004)`); err != nil {
		t.Fatalf("seed entities: %v", err)
	}
	if _, err := store.DB().Exec(`INSERT INTO relationships(source_id, target_id, predicate, updated_at) VALUES
		(1, 2, 'RELATED_TO', 1710000005)`); err != nil {
		t.Fatalf("seed relationships: %v", err)
	}
	if _, err := store.DB().Exec(`INSERT INTO goncho_memory_items(
		memory_id, workspace_id, agent_id, observer_peer_id, peer_id, scope, content, source_kind, importance, active, created_at, updated_at
	) VALUES
		('mem-active', 'gormes', 'mineru', 'gormes', 'mineru', 'private', 'Active memory', 'manual', 0.9, 1, 1710000006, 1710000006),
		('mem-archived', 'gormes', 'mineru', 'gormes', 'mineru', 'private', 'Old memory', 'manual', 0.4, 0, 1710000007, 1710000007)`); err != nil {
		t.Fatalf("seed memory items: %v", err)
	}
	if _, err := store.DB().Exec(`INSERT INTO goncho_conclusions(
		workspace_id, observer_peer_id, peer_id, session_key, content, kind, status, source, idempotency_key, created_at, updated_at
	) VALUES ('gormes', 'gormes', 'mineru', 's-1', 'Mineru uses Goncho memory', 'fact', 'processed', 'manual', 'idem-1', 1710000008, 1710000008)`); err != nil {
		t.Fatalf("seed conclusion: %v", err)
	}
	if _, err := store.DB().Exec(`INSERT INTO goncho_session_summaries(
		workspace_id, session_key, summary_type, content, message_id, created_at, token_count
	) VALUES ('gormes', 's-1', 'short', 'Session summary', 2, 1710000009, 12)`); err != nil {
		t.Fatalf("seed summary: %v", err)
	}

	prev := navivoxMemoryDBPath
	navivoxMemoryDBPath = func() string { return memoryDB }
	t.Cleanup(func() { navivoxMemoryDBPath = prev })

	ch := newTestChannel(t)
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()

	unauth, err := http.Get(server.URL + "/v1/navivox/memory/overview?server_id=local&profile_id=mineru")
	if err != nil {
		t.Fatal(err)
	}
	defer unauth.Body.Close()
	if unauth.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, want 401", unauth.StatusCode)
	}

	req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/navivox/memory/overview?server_id=local&profile_id=mineru", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer nvbx_test_token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("authorized status = %d, want 200", resp.StatusCode)
	}

	var payload struct {
		ProfileID     string         `json:"profile_id"`
		WorkspaceID   string         `json:"workspace_id"`
		DatabaseLabel string         `json:"database_label"`
		Health        string         `json:"health"`
		Counts        map[string]int `json:"counts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.ProfileID != "mineru" || payload.WorkspaceID != "gormes" || payload.Health != "active" {
		t.Fatalf("overview identity = %+v", payload)
	}
	if payload.DatabaseLabel == memoryDB || filepath.IsAbs(payload.DatabaseLabel) {
		t.Fatalf("database_label = %q, want redacted non-absolute label", payload.DatabaseLabel)
	}
	wantCounts := map[string]int{
		"turns":             2,
		"memory_items":      1,
		"observations":      0,
		"conclusions":       1,
		"session_summaries": 1,
		"entities":          2,
		"relationships":     1,
	}
	for key, want := range wantCounts {
		if got := payload.Counts[key]; got != want {
			t.Fatalf("counts[%s] = %d, want %d (payload=%+v)", key, got, want, payload.Counts)
		}
	}
}
