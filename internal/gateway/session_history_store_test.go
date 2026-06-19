package gateway

import (
	"context"
	"database/sql"
	"testing"

	"log/slog"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

func TestSQLSessionHistoryStoreRewriteDeletesPersistedTail(t *testing.T) {
	ctx := context.Background()
	store, db := openTestSessionHistoryStore(t)
	seedSQLHistoryTurns(t, db, "sess-retry", []llm.Message{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "first answer"},
		{Role: "user", Content: "retry me"},
		{Role: "assistant", Content: "bad answer"},
	})

	if err := store.RewriteSessionHistory(ctx, "sess-retry", []llm.Message{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "first answer"},
	}); err != nil {
		t.Fatalf("RewriteSessionHistory: %v", err)
	}
	got, err := store.LoadSessionHistory(ctx, "sess-retry")
	if err != nil {
		t.Fatalf("LoadSessionHistory: %v", err)
	}
	assertHistoryMessages(t, got, []llm.Message{{Role: "user", Content: "first"}, {Role: "assistant", Content: "first answer"}})
	assertSQLTurnCount(t, db, "sess-retry", 2)
}

func TestSQLSessionHistoryStoreRewindDeletesFromNthUserTurn(t *testing.T) {
	ctx := context.Background()
	store, db := openTestSessionHistoryStore(t)
	seedSQLHistoryTurns(t, db, "sess-undo", []llm.Message{
		{Role: "user", Content: "one"},
		{Role: "assistant", Content: "one answer"},
		{Role: "user", Content: "two"},
		{Role: "assistant", Content: "two answer"},
		{Role: "user", Content: "three"},
		{Role: "assistant", Content: "three answer"},
	})

	result, err := store.RewindSessionHistory(ctx, "sess-undo", 2)
	if err != nil {
		t.Fatalf("RewindSessionHistory: %v", err)
	}
	if result.SessionID != "sess-undo" || result.TargetText != "two" || result.TurnsUndone != 2 || result.RewoundCount != 4 {
		t.Fatalf("result = %+v", result)
	}
	assertHistoryMessages(t, result.History, []llm.Message{{Role: "user", Content: "one"}, {Role: "assistant", Content: "one answer"}})
	got, err := store.LoadSessionHistory(ctx, "sess-undo")
	if err != nil {
		t.Fatalf("LoadSessionHistory: %v", err)
	}
	assertHistoryMessages(t, got, result.History)
	assertSQLTurnCount(t, db, "sess-undo", 2)
}

func TestGatewayCommandDispatchRetryUsesSQLSessionHistoryStore(t *testing.T) {
	ctx := context.Background()
	store, db := openTestSessionHistoryStore(t)
	seedSQLHistoryTurns(t, db, "sess-retry-command", []llm.Message{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "first answer"},
		{Role: "user", Content: "retry durable"},
		{Role: "assistant", Content: "bad answer"},
	})
	smap := session.NewMemMap()
	if err := smap.Put(ctx, "telegram:42", "sess-retry-command"); err != nil {
		t.Fatalf("session map put: %v", err)
	}
	ch := newFakeChannel("telegram")
	fk := &fakeKernel{}
	m := NewManagerWithSubmitter(ManagerConfig{AllowedChats: map[string]string{"telegram": "42"}, SessionMap: smap, SessionHistoryStore: store}, fk, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := m.handleInbound(ctx, InboundEvent{Platform: "telegram", ChatID: "42", MsgID: "retry-msg", Kind: EventRetry, Text: "/retry"}); err != nil {
		t.Fatalf("handle retry: %v", err)
	}
	resumes := fk.resumesSnapshot()
	if len(resumes) != 1 || resumes[0].SessionID != "sess-retry-command" || len(resumes[0].History) != 2 {
		t.Fatalf("resumes = %+v", resumes)
	}
	submits := fk.submitsSnapshot()
	if len(submits) != 1 || submits[0].Text != "retry durable" || submits[0].SessionID != "sess-retry-command" {
		t.Fatalf("submits = %+v", submits)
	}
	got, err := store.LoadSessionHistory(ctx, "sess-retry-command")
	if err != nil {
		t.Fatalf("LoadSessionHistory: %v", err)
	}
	assertHistoryMessages(t, got, []llm.Message{{Role: "user", Content: "first"}, {Role: "assistant", Content: "first answer"}})
}

func TestGatewayCommandDispatchUndoUsesSQLSessionHistoryStore(t *testing.T) {
	ctx := context.Background()
	store, db := openTestSessionHistoryStore(t)
	seedSQLHistoryTurns(t, db, "sess-undo-command", []llm.Message{
		{Role: "user", Content: "one"},
		{Role: "assistant", Content: "one answer"},
		{Role: "user", Content: "two"},
		{Role: "assistant", Content: "two answer"},
	})
	smap := session.NewMemMap()
	if err := smap.Put(ctx, "telegram:42", "sess-undo-command"); err != nil {
		t.Fatalf("session map put: %v", err)
	}
	ch := newFakeChannel("telegram")
	fk := &fakeKernel{}
	m := NewManagerWithSubmitter(ManagerConfig{AllowedChats: map[string]string{"telegram": "42"}, SessionMap: smap, SessionHistoryStore: store}, fk, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if err := m.handleInbound(ctx, InboundEvent{Platform: "telegram", ChatID: "42", MsgID: "undo-msg", Kind: EventUndo, Text: "/undo"}); err != nil {
		t.Fatalf("handle undo: %v", err)
	}
	resumes := fk.resumesSnapshot()
	if len(resumes) != 1 || resumes[0].SessionID != "sess-undo-command" || len(resumes[0].History) != 2 {
		t.Fatalf("resumes = %+v", resumes)
	}
	sent := ch.sentSnapshot()
	if len(sent) == 0 || !strings.Contains(sent[len(sent)-1].Text, "Removed 1 turn") || !strings.Contains(sent[len(sent)-1].Text, "two") {
		t.Fatalf("sent = %+v", sent)
	}
	got, err := store.LoadSessionHistory(ctx, "sess-undo-command")
	if err != nil {
		t.Fatalf("LoadSessionHistory: %v", err)
	}
	assertHistoryMessages(t, got, []llm.Message{{Role: "user", Content: "one"}, {Role: "assistant", Content: "one answer"}})
}

func openTestSessionHistoryStore(t *testing.T) (SessionHistoryStore, *sql.DB) {
	t.Helper()
	mstore, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	t.Cleanup(func() { _ = mstore.Close(context.Background()) })
	return NewSQLSessionHistoryStore(mstore.DB()), mstore.DB()
}

func seedSQLHistoryTurns(t *testing.T, db *sql.DB, sessionID string, messages []llm.Message) {
	t.Helper()
	for i, msg := range messages {
		if _, err := db.Exec(`INSERT INTO turns(session_id, role, content, ts_unix, memory_sync_status) VALUES(?, ?, ?, ?, 'ready')`, sessionID, msg.Role, msg.Content, 100+i); err != nil {
			t.Fatalf("seed turn %d: %v", i, err)
		}
	}
}

func assertHistoryMessages(t *testing.T, got, want []llm.Message) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("history len = %d, want %d; got=%+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Role != want[i].Role || got[i].Content != want[i].Content {
			t.Fatalf("history[%d] = %+v, want %+v; full=%+v", i, got[i], want[i], got)
		}
	}
}

func assertSQLTurnCount(t *testing.T, db *sql.DB, sessionID string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(`SELECT COUNT(*) FROM turns WHERE session_id = ?`, sessionID).Scan(&got); err != nil {
		t.Fatalf("count turns: %v", err)
	}
	if got != want {
		t.Fatalf("turn count = %d, want %d", got, want)
	}
}
