package memory

import (
	"context"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
)

func TestSessionCatalog_SearchHitsIncludeLineageContext(t *testing.T) {
	store, err := OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer func() {
		if err := store.Close(context.Background()); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()

	ctx := context.Background()
	for _, turn := range []struct {
		sessionID string
		chatID    string
		content   string
		ts        int64
	}{
		{
			sessionID: "sess-root",
			chatID:    "telegram:42",
			content:   "Atlas root evidence",
			ts:        100,
		},
		{
			sessionID: "sess-child",
			chatID:    "telegram:42",
			content:   "Atlas compressed continuation",
			ts:        200,
		},
		{
			sessionID: "sess-orphan",
			chatID:    "telegram:42",
			content:   "Atlas orphaned branch",
			ts:        300,
		},
		{
			sessionID: "sess-chat-only",
			chatID:    "telegram:42",
			content:   "Atlas legacy row matched only by chat key",
			ts:        400,
		},
	} {
		if _, err := store.DB().ExecContext(ctx,
			`INSERT INTO turns(session_id, role, content, ts_unix, chat_id) VALUES (?, ?, ?, ?, ?)`,
			turn.sessionID, "user", turn.content, turn.ts, turn.chatID,
		); err != nil {
			t.Fatalf("insert turn %s: %v", turn.sessionID, err)
		}
	}

	metas := []session.Metadata{
		{SessionID: "sess-root", Source: "telegram", ChatID: "42", UserID: "user-juan"},
		{
			SessionID:       "sess-child",
			Source:          "telegram",
			ChatID:          "42",
			UserID:          "user-juan",
			ParentSessionID: "sess-root",
			LineageKind:     session.LineageKindCompression,
		},
		{
			SessionID:       "sess-orphan",
			Source:          "telegram",
			ChatID:          "42",
			UserID:          "user-juan",
			ParentSessionID: "sess-missing",
			LineageKind:     session.LineageKindFork,
		},
	}

	messages, err := SearchMessages(ctx, store.DB(), metas, SearchFilter{
		UserID:  "user-juan",
		Sources: []string{"telegram"},
		Query:   "Atlas",
	}, 10)
	if err != nil {
		t.Fatalf("SearchMessages: %v", err)
	}
	if len(messages) != 4 {
		t.Fatalf("SearchMessages len = %d, want 4", len(messages))
	}
	messageBySession := make(map[string]MessageSearchHit, len(messages))
	for _, hit := range messages {
		messageBySession[hit.SessionID] = hit
	}
	assertSearchLineage(t, "message root", messageBySession["sess-root"].Lineage, SearchLineage{
		LineageKind:     session.LineageKindPrimary,
		ChildSessionIDs: []string{"sess-child"},
		Status:          session.LineageStatusOK,
	})
	assertSearchLineage(t, "message child", messageBySession["sess-child"].Lineage, SearchLineage{
		ParentSessionID: "sess-root",
		LineageKind:     session.LineageKindCompression,
		Status:          session.LineageStatusOK,
	})
	assertSearchLineage(t, "message orphan", messageBySession["sess-orphan"].Lineage, SearchLineage{
		ParentSessionID: "sess-missing",
		LineageKind:     session.LineageKindFork,
		Status:          session.LineageStatusOrphan,
	})
	assertSearchLineage(t, "message chat-only", messageBySession["sess-chat-only"].Lineage, SearchLineage{
		Status: SearchLineageStatusUnavailable,
	})

	sessions, err := SearchSessions(ctx, store.DB(), metas, SearchFilter{
		UserID:  "user-juan",
		Sources: []string{"telegram"},
		Query:   "Atlas",
	}, 10)
	if err != nil {
		t.Fatalf("SearchSessions: %v", err)
	}
	if len(sessions) != 4 {
		t.Fatalf("SearchSessions len = %d, want 4", len(sessions))
	}
	sessionByID := make(map[string]SessionSearchHit, len(sessions))
	for _, hit := range sessions {
		sessionByID[hit.SessionID] = hit
	}
	assertSearchLineage(t, "session root", sessionByID["sess-root"].Lineage, SearchLineage{
		LineageKind:     session.LineageKindPrimary,
		ChildSessionIDs: []string{"sess-child"},
		Status:          session.LineageStatusOK,
	})
	assertSearchLineage(t, "session child", sessionByID["sess-child"].Lineage, SearchLineage{
		ParentSessionID: "sess-root",
		LineageKind:     session.LineageKindCompression,
		Status:          session.LineageStatusOK,
	})
	assertSearchLineage(t, "session orphan", sessionByID["sess-orphan"].Lineage, SearchLineage{
		ParentSessionID: "sess-missing",
		LineageKind:     session.LineageKindFork,
		Status:          session.LineageStatusOrphan,
	})
	assertSearchLineage(t, "session chat-only", sessionByID["sess-chat-only"].Lineage, SearchLineage{
		Status: SearchLineageStatusUnavailable,
	})
}

func TestSessionCatalog_LineageMetadataDoesNotWidenDefaultRecall(t *testing.T) {
	_, p := openProviderWithRichGraph(t)
	dir := session.NewMemMap()
	ctx := context.Background()
	if err := dir.PutMetadata(ctx, session.Metadata{
		SessionID: "s",
		Source:    "telegram",
		ChatID:    "42",
		UserID:    "user-juan",
	}); err != nil {
		t.Fatalf("PutMetadata root: %v", err)
	}
	if err := dir.PutMetadata(ctx, session.Metadata{
		SessionID:       "s-compressed",
		Source:          "telegram",
		ChatID:          "42",
		UserID:          "user-juan",
		ParentSessionID: "s",
		LineageKind:     session.LineageKindCompression,
	}); err != nil {
		t.Fatalf("PutMetadata child: %v", err)
	}

	p = p.WithDirectory(dir)
	out := p.GetContext(ctx, RecallInput{
		UserMessage: "Acme progress?",
		ChatKey:     "discord:7",
		UserID:      "user-juan",
	})
	if out != "" {
		t.Fatalf("default same-chat recall widened through lineage metadata; got %q", out)
	}
}

func assertSearchLineage(t *testing.T, label string, got, want SearchLineage) {
	t.Helper()
	if got.ParentSessionID != want.ParentSessionID ||
		got.LineageKind != want.LineageKind ||
		got.Status != want.Status {
		t.Fatalf("%s lineage = %+v, want parent %q kind %q status %q",
			label, got, want.ParentSessionID, want.LineageKind, want.Status)
	}
	if len(got.ChildSessionIDs) != len(want.ChildSessionIDs) {
		t.Fatalf("%s children = %v, want %v", label, got.ChildSessionIDs, want.ChildSessionIDs)
	}
	for i := range want.ChildSessionIDs {
		if got.ChildSessionIDs[i] != want.ChildSessionIDs[i] {
			t.Fatalf("%s children = %v, want %v", label, got.ChildSessionIDs, want.ChildSessionIDs)
		}
	}
}
