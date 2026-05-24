package sessionsearch

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
)

func TestSessionSearchToolExecution_SameChatDefault(t *testing.T) {
	ctx := context.Background()
	store, dir := newSessionSearchFixture(t)
	seedSessionSearchMetadata(t, ctx, dir,
		session.Metadata{SessionID: "sess-current", Source: "discord", ChatID: "chan-1", UserID: "user-juan", UpdatedAt: 30},
		session.Metadata{SessionID: "sess-same-chat", Source: "discord", ChatID: "chan-1", UserID: "user-juan", UpdatedAt: 20},
		session.Metadata{SessionID: "sess-cross-chat", Source: "telegram", ChatID: "42", UserID: "user-juan", UpdatedAt: 10},
	)
	seedSessionSearchTurns(t, ctx, store,
		sessionSearchTurn{sessionID: "sess-same-chat", chatID: "discord:chan-1", content: "Atlas same chat detail", ts: 200},
		sessionSearchTurn{sessionID: "sess-cross-chat", chatID: "telegram:42", content: "Atlas cross chat detail", ts: 300},
	)

	tool := NewSessionSearchTool(SessionSearchToolConfig{
		DB:       store.DB(),
		Sessions: dir,
	})
	payload := executeSessionSearchTool(t, tool, `{"query":"Atlas","current_session_id":"sess-current","limit":5}`)

	if !payload.Success {
		t.Fatalf("Success = false, evidence = %+v", payload.Evidence)
	}
	if got := sessionSearchResultIDs(payload.Results); !slices.Equal(got, []string{"sess-same-chat"}) {
		t.Fatalf("results session IDs = %v, want same-chat only", got)
	}
}

func TestSessionSearchToolExecution_UserScopeSourceFilter(t *testing.T) {
	ctx := context.Background()
	store, dir := newSessionSearchFixture(t)
	seedSessionSearchMetadata(t, ctx, dir,
		session.Metadata{SessionID: "sess-current", Source: "discord", ChatID: "chan-1", UserID: "user-juan", UpdatedAt: 30},
		session.Metadata{SessionID: "sess-telegram", Source: "telegram", ChatID: "42", UserID: "user-juan", UpdatedAt: 20},
		session.Metadata{SessionID: "sess-slack", Source: "slack", ChatID: "C123", UserID: "user-juan", UpdatedAt: 10},
	)
	seedSessionSearchTurns(t, ctx, store,
		sessionSearchTurn{sessionID: "sess-telegram", chatID: "telegram:42", content: "Atlas telegram detail", ts: 200},
		sessionSearchTurn{sessionID: "sess-slack", chatID: "slack:C123", content: "Atlas slack detail", ts: 300},
	)

	tool := NewSessionSearchTool(SessionSearchToolConfig{
		DB:       store.DB(),
		Sessions: dir,
	})
	payload := executeSessionSearchTool(t, tool, `{"query":"Atlas","scope":"user","sources":["telegram"],"current_session_id":"sess-current","limit":5}`)

	if !payload.Success {
		t.Fatalf("Success = false, evidence = %+v", payload.Evidence)
	}
	if got := sessionSearchResultIDs(payload.Results); !slices.Equal(got, []string{"sess-telegram"}) {
		t.Fatalf("results session IDs = %v, want telegram source only", got)
	}
	if payload.Results[0].OriginSource != "telegram" {
		t.Fatalf("origin source = %q, want telegram evidence", payload.Results[0].OriginSource)
	}
	if payload.ScopeEvidence == nil {
		t.Fatal("scope_evidence = nil, want Goncho/Honcho-compatible widening evidence")
	}
}

func TestSessionSearchToolExecution_RoleFilter(t *testing.T) {
	ctx := context.Background()
	store, dir := newSessionSearchFixture(t)
	seedSessionSearchMetadata(t, ctx, dir,
		session.Metadata{SessionID: "sess-current", Source: "discord", ChatID: "chan-1", UserID: "user-juan", UpdatedAt: 30},
		session.Metadata{SessionID: "sess-same-chat", Source: "discord", ChatID: "chan-1", UserID: "user-juan", UpdatedAt: 20},
	)
	if _, err := store.DB().ExecContext(ctx,
		`INSERT INTO turns(session_id, role, content, ts_unix, chat_id)
		 VALUES
		 ('sess-same-chat', 'assistant', 'Atlas assistant-only detail', 300, 'discord:chan-1'),
		 ('sess-same-chat', 'user', 'Atlas user-only detail', 200, 'discord:chan-1')`,
	); err != nil {
		t.Fatal(err)
	}

	tool := NewSessionSearchTool(SessionSearchToolConfig{
		DB:       store.DB(),
		Sessions: dir,
	})
	payload := executeSessionSearchTool(t, tool, `{"query":"Atlas","role_filter":"user","current_session_id":"sess-current","limit":5}`)

	if !payload.Success {
		t.Fatalf("Success = false, evidence = %+v", payload.Evidence)
	}
	if got := sessionSearchResultRoles(payload.Results); !slices.Equal(got, []string{"user"}) {
		t.Fatalf("result roles = %v, want user only", got)
	}
	if payload.Results[0].Content != "Atlas user-only detail" {
		t.Fatalf("content = %q, want user turn only", payload.Results[0].Content)
	}
}

func TestSessionSearchToolExecution_HidesToolSourceByDefault(t *testing.T) {
	ctx := context.Background()
	store, dir := newSessionSearchFixture(t)
	seedSessionSearchMetadata(t, ctx, dir,
		session.Metadata{SessionID: "sess-current", Source: "discord", ChatID: "chan-1", UserID: "user-juan", UpdatedAt: 40},
		session.Metadata{SessionID: "sess-tool", Source: "tool", ChatID: "paperclip-1", UserID: "user-juan", UpdatedAt: 30},
		session.Metadata{SessionID: "sess-telegram", Source: "telegram", ChatID: "42", UserID: "user-juan", UpdatedAt: 20},
	)
	seedSessionSearchTurns(t, ctx, store,
		sessionSearchTurn{sessionID: "sess-tool", chatID: "tool:paperclip-1", content: "Atlas tool-side detail", ts: 300},
		sessionSearchTurn{sessionID: "sess-telegram", chatID: "telegram:42", content: "Atlas telegram detail", ts: 200},
	)
	tool := NewSessionSearchTool(SessionSearchToolConfig{
		DB:       store.DB(),
		Sessions: dir,
	})

	defaultPayload := executeSessionSearchTool(t, tool, `{"query":"Atlas","scope":"user","current_session_id":"sess-current","limit":5}`)
	if !defaultPayload.Success {
		t.Fatalf("Success = false, evidence = %+v", defaultPayload.Evidence)
	}
	if got := sessionSearchResultIDs(defaultPayload.Results); !slices.Equal(got, []string{"sess-telegram"}) {
		t.Fatalf("default source results = %v, want tool source hidden", got)
	}

	explicitPayload := executeSessionSearchTool(t, tool, `{"query":"Atlas","scope":"user","sources":["tool"],"current_session_id":"sess-current","limit":5}`)
	if !explicitPayload.Success {
		t.Fatalf("explicit tool source Success = false, evidence = %+v", explicitPayload.Evidence)
	}
	if got := sessionSearchResultIDs(explicitPayload.Results); !slices.Equal(got, []string{"sess-tool"}) {
		t.Fatalf("explicit tool source results = %v, want tool source when requested", got)
	}
}

func TestSessionSearchToolExecution_ResultMessages(t *testing.T) {
	ctx := context.Background()
	store, dir := newSessionSearchFixture(t)
	seedSessionSearchMetadata(t, ctx, dir,
		session.Metadata{SessionID: "sess-current", Source: "discord", ChatID: "chan-1", UserID: "user-juan", UpdatedAt: 30},
		session.Metadata{SessionID: "sess-other", Source: "discord", ChatID: "chan-1", UserID: "user-juan", UpdatedAt: 20},
	)
	seedSessionSearchTurns(t, ctx, store,
		sessionSearchTurn{sessionID: "sess-current", chatID: "discord:chan-1", content: "current conversation", ts: 300},
		sessionSearchTurn{sessionID: "sess-other", chatID: "discord:chan-1", content: "older conversation", ts: 200},
	)
	tool := NewSessionSearchTool(SessionSearchToolConfig{
		DB:       store.DB(),
		Sessions: dir,
	})

	empty := executeSessionSearchTool(t, tool, `{"query":"NoSuchNeedle","current_session_id":"sess-current","limit":5}`)
	if !empty.Success || empty.Count != 0 || empty.Message != "No matching sessions found." {
		t.Fatalf("empty result = %+v, want Hermes no-match message", empty)
	}

	recent := executeSessionSearchTool(t, tool, `{"mode":"recent","current_session_id":"sess-current","limit":5}`)
	if !recent.Success || recent.Count != 1 || recent.Message != "Showing 1 most recent sessions. Use a keyword query to search specific topics." {
		t.Fatalf("recent result = %+v, want Hermes recent-mode message", recent)
	}
}

func TestSessionSearchToolExecution_RecentModeExcludesCurrentLineageRoot(t *testing.T) {
	ctx := context.Background()
	store, dir := newSessionSearchFixture(t)
	seedSessionSearchMetadata(t, ctx, dir,
		session.Metadata{SessionID: "sess-root", Source: "telegram", ChatID: "42", UserID: "user-juan", UpdatedAt: 30},
		session.Metadata{
			SessionID:       "sess-child",
			Source:          "telegram",
			ChatID:          "42",
			UserID:          "user-juan",
			ParentSessionID: "sess-root",
			LineageKind:     session.LineageKindCompression,
			UpdatedAt:       40,
		},
		session.Metadata{SessionID: "sess-other", Source: "telegram", ChatID: "42", UserID: "user-juan", UpdatedAt: 20},
	)
	seedSessionSearchTurns(t, ctx, store,
		sessionSearchTurn{sessionID: "sess-root", chatID: "telegram:42", content: "root conversation", ts: 200},
		sessionSearchTurn{sessionID: "sess-child", chatID: "telegram:42", content: "compressed child conversation", ts: 300},
		sessionSearchTurn{sessionID: "sess-other", chatID: "telegram:42", content: "other conversation", ts: 100},
	)

	tool := NewSessionSearchTool(SessionSearchToolConfig{
		DB:       store.DB(),
		Sessions: dir,
	})
	payload := executeSessionSearchTool(t, tool, `{"mode":"recent","current_session_id":"sess-child","limit":5}`)

	if !payload.Success {
		t.Fatalf("Success = false, evidence = %+v", payload.Evidence)
	}
	if got := sessionSearchResultIDs(payload.Results); !slices.Equal(got, []string{"sess-other"}) {
		t.Fatalf("results session IDs = %v, want current lineage excluded", got)
	}
	if !sessionSearchHasEvidence(payload.Evidence, "lineage_root_excluded") {
		t.Fatalf("evidence = %+v, want lineage_root_excluded", payload.Evidence)
	}
}

func TestSessionSearchToolExecution_DegradedEvidence(t *testing.T) {
	t.Run("missing session directory", func(t *testing.T) {
		store, _ := newSessionSearchFixture(t)
		tool := NewSessionSearchTool(SessionSearchToolConfig{DB: store.DB()})

		payload := executeSessionSearchTool(t, tool, `{"query":"Atlas","current_session_id":"sess-current"}`)

		if payload.Success {
			t.Fatalf("Success = true, want degraded unavailable result")
		}
		if !sessionSearchHasEvidence(payload.Evidence, "session_search_unavailable") {
			t.Fatalf("evidence = %+v, want session_search_unavailable", payload.Evidence)
		}
	})

	t.Run("denied source widening", func(t *testing.T) {
		ctx := context.Background()
		store, dir := newSessionSearchFixture(t)
		seedSessionSearchMetadata(t, ctx, dir,
			session.Metadata{SessionID: "sess-current", Source: "discord", ChatID: "chan-1", UserID: "user-juan", UpdatedAt: 30},
			session.Metadata{SessionID: "sess-telegram", Source: "telegram", ChatID: "42", UserID: "user-juan", UpdatedAt: 20},
		)
		seedSessionSearchTurns(t, ctx, store,
			sessionSearchTurn{sessionID: "sess-telegram", chatID: "telegram:42", content: "Atlas telegram detail", ts: 200},
		)
		tool := NewSessionSearchTool(SessionSearchToolConfig{
			DB:       store.DB(),
			Sessions: dir,
		})

		payload := executeSessionSearchTool(t, tool, `{"query":"Atlas","sources":["telegram"],"current_session_id":"sess-current","limit":5}`)

		if payload.Success {
			t.Fatalf("Success = true, want denied source widening")
		}
		if !sessionSearchHasEvidence(payload.Evidence, "source_filter_denied") {
			t.Fatalf("evidence = %+v, want source_filter_denied", payload.Evidence)
		}
		if len(payload.Results) != 0 {
			t.Fatalf("results = %+v, want no hidden fallback widening", payload.Results)
		}
	})
}

type sessionSearchExecutionPayload struct {
	Success       bool                    `json:"success"`
	Count         int                     `json:"count"`
	Message       string                  `json:"message"`
	Results       []sessionSearchHit      `json:"results"`
	Evidence      []SessionSearchEvidence `json:"evidence"`
	ScopeEvidence any                     `json:"scope_evidence"`
}

type sessionSearchHit struct {
	SessionID    string `json:"session_id"`
	Source       string `json:"source"`
	OriginSource string `json:"origin_source"`
	Role         string `json:"role"`
	Content      string `json:"content"`
}

type sessionSearchTurn struct {
	sessionID string
	chatID    string
	content   string
	ts        int64
}

func newSessionSearchFixture(t *testing.T) (*memory.SqliteStore, *session.MemMap) {
	t.Helper()
	store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(context.Background()); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	return store, session.NewMemMap()
}

func seedSessionSearchMetadata(t *testing.T, ctx context.Context, dir *session.MemMap, metas ...session.Metadata) {
	t.Helper()
	for _, meta := range metas {
		if err := dir.PutMetadata(ctx, meta); err != nil {
			t.Fatalf("PutMetadata(%s): %v", meta.SessionID, err)
		}
	}
}

func seedSessionSearchTurns(t *testing.T, ctx context.Context, store *memory.SqliteStore, turns ...sessionSearchTurn) {
	t.Helper()
	for _, turn := range turns {
		if _, err := store.DB().ExecContext(ctx,
			`INSERT INTO turns(session_id, role, content, ts_unix, chat_id) VALUES (?, 'user', ?, ?, ?)`,
			turn.sessionID, turn.content, turn.ts, turn.chatID,
		); err != nil {
			t.Fatalf("insert turn %s: %v", turn.sessionID, err)
		}
	}
}

func executeSessionSearchTool(t *testing.T, tool *SessionSearchTool, raw string) sessionSearchExecutionPayload {
	t.Helper()
	out, err := tool.Execute(context.Background(), json.RawMessage(raw))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var payload sessionSearchExecutionPayload
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("Execute output invalid JSON: %s: %v", out, err)
	}
	return payload
}

func sessionSearchResultIDs(results []sessionSearchHit) []string {
	out := make([]string, 0, len(results))
	for _, result := range results {
		out = append(out, result.SessionID)
	}
	return out
}

func sessionSearchResultRoles(results []sessionSearchHit) []string {
	out := make([]string, 0, len(results))
	for _, result := range results {
		out = append(out, result.Role)
	}
	return out
}

func sessionSearchHasEvidence(items []SessionSearchEvidence, status string) bool {
	for _, item := range items {
		if item.Status == status {
			return true
		}
	}
	return false
}
