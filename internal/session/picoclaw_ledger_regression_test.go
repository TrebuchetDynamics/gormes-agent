package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/transcript"
)

func TestPicoClawSessionLedger_MultipleUserMessagesRemainVisible(t *testing.T) {
	db := openPicoClawLedgerTestDB(t)
	seedPicoClawLedgerTurn(t, db, picoclawLedgerTurn{SessionID: "sess-history", Role: "user", Content: "first user request", TSUnix: 100, ChatID: "telegram:42", MetaJSON: senderMeta(t, "user-1", "Alice")})
	seedPicoClawLedgerTurn(t, db, picoclawLedgerTurn{SessionID: "sess-history", Role: "assistant", Content: "first answer", TSUnix: 101, ChatID: "telegram:42"})
	seedPicoClawLedgerTurn(t, db, picoclawLedgerTurn{SessionID: "sess-history", Role: "user", Content: "second user request", TSUnix: 200, ChatID: "telegram:42", MetaJSON: senderMeta(t, "user-1", "Alice")})
	seedPicoClawLedgerTurn(t, db, picoclawLedgerTurn{SessionID: "sess-history", Role: "assistant", Content: "second answer", TSUnix: 201, ChatID: "telegram:42"})

	ledger, err := ReadSessionLedger(context.Background(), db, nil, "sess-history")
	if err != nil {
		t.Fatalf("ReadSessionLedger: %v", err)
	}
	if len(ledger.Messages) != 4 {
		t.Fatalf("ledger messages = %d, want 4", len(ledger.Messages))
	}
	gotUsers := []string{}
	for _, msg := range ledger.Messages {
		if msg.Role == "user" {
			gotUsers = append(gotUsers, msg.Content)
			if msg.SenderID != "user-1" || msg.SenderName != "Alice" {
				t.Fatalf("user sender attribution = (%q, %q), want Alice/user-1", msg.SenderID, msg.SenderName)
			}
		}
	}
	if strings.Join(gotUsers, "|") != "first user request|second user request" {
		t.Fatalf("user messages = %#v, want both user requests in order", gotUsers)
	}

	md, err := transcript.ExportMarkdown(context.Background(), db, "sess-history")
	if err != nil {
		t.Fatalf("ExportMarkdown: %v", err)
	}
	first := strings.Index(md, "first user request")
	second := strings.Index(md, "second user request")
	if first < 0 || second < 0 || first > second {
		t.Fatalf("markdown did not preserve both user messages in order:\n%s", md)
	}
}

func TestPicoClawSessionLedger_PerMessageTimestampsDoNotUseSessionUpdated(t *testing.T) {
	db := openPicoClawLedgerTestDB(t)
	meta := NewMemMap()
	if err := meta.PutMetadata(context.Background(), Metadata{
		SessionID: "sess-time",
		Source:    "telegram",
		ChatID:    "42",
		UserID:    "user-1",
		CreatedAt: 1_000,
		UpdatedAt: 9_999,
	}); err != nil {
		t.Fatalf("PutMetadata: %v", err)
	}
	seedPicoClawLedgerTurn(t, db, picoclawLedgerTurn{SessionID: "sess-time", Role: "user", Content: "23:01 question", TSUnix: 1_000, ChatID: "telegram:42"})
	seedPicoClawLedgerTurn(t, db, picoclawLedgerTurn{SessionID: "sess-time", Role: "assistant", Content: "23:01 answer", TSUnix: 1_001, ChatID: "telegram:42"})
	seedPicoClawLedgerTurn(t, db, picoclawLedgerTurn{SessionID: "sess-time", Role: "user", Content: "23:48 question", TSUnix: 3_820, ChatID: "telegram:42"})

	ledger, err := ReadSessionLedger(context.Background(), db, meta, "sess-time")
	if err != nil {
		t.Fatalf("ReadSessionLedger: %v", err)
	}
	if ledger.UpdatedAtUnix != 9_999 {
		t.Fatalf("ledger UpdatedAtUnix = %d, want session metadata updated timestamp", ledger.UpdatedAtUnix)
	}
	got := []int64{ledger.Messages[0].CreatedAtUnix, ledger.Messages[1].CreatedAtUnix, ledger.Messages[2].CreatedAtUnix}
	if want := []int64{1_000, 1_001, 3_820}; !reflect.DeepEqual(got, want) {
		t.Fatalf("message timestamps = %#v, want per-message ts_unix values, not session.updated", got)
	}
	for _, msg := range ledger.Messages {
		if msg.CreatedAtUnix == ledger.UpdatedAtUnix {
			t.Fatalf("message %d used session.updated as timestamp: %+v", msg.ID, msg)
		}
	}

	seedPicoClawLedgerTurn(t, db, picoclawLedgerTurn{SessionID: "sess-legacy", Role: "user", Content: "legacy without per-message timestamp", TSUnix: 0})
	legacy, err := ReadSessionLedger(context.Background(), db, nil, "sess-legacy")
	if err != nil {
		t.Fatalf("ReadSessionLedger legacy: %v", err)
	}
	if len(legacy.Messages) != 1 || legacy.Messages[0].CreatedAtKnown {
		t.Fatalf("legacy timestamp evidence = %+v, want unknown timestamp", legacy.Messages)
	}
	if legacy.Messages[0].TimestampEvidence != "timestamp_unknown_legacy" {
		t.Fatalf("legacy timestamp evidence = %q, want timestamp_unknown_legacy", legacy.Messages[0].TimestampEvidence)
	}
	md, err := transcript.ExportMarkdown(context.Background(), db, "sess-legacy")
	if err != nil {
		t.Fatalf("ExportMarkdown legacy: %v", err)
	}
	if !strings.Contains(md, "unknown timestamp") || strings.Contains(md, "1970-01-01") {
		t.Fatalf("legacy markdown timestamp not visibly degraded:\n%s", md)
	}
}

func TestPicoClawSessionLedger_ResetBoundaryIsNonDestructive(t *testing.T) {
	ctx := context.Background()
	db := openPicoClawLedgerTestDB(t)
	meta := NewMemMap()
	if err := meta.PutMetadata(ctx, Metadata{SessionID: "sess-root", Source: "telegram", ChatID: "42", UserID: "user-1", CreatedAt: 100, UpdatedAt: 200}); err != nil {
		t.Fatalf("PutMetadata root: %v", err)
	}
	if err := meta.PutMetadata(ctx, Metadata{SessionID: "sess-fresh", Source: "telegram", ChatID: "42", UserID: "user-1", ParentSessionID: "sess-root", LineageKind: LineageKindCompression, CreatedAt: 300, UpdatedAt: 400}); err != nil {
		t.Fatalf("PutMetadata fresh: %v", err)
	}
	seedPicoClawLedgerTurn(t, db, picoclawLedgerTurn{SessionID: "sess-root", Role: "user", Content: "old context question", TSUnix: 100, ChatID: "telegram:42"})
	seedPicoClawLedgerTurn(t, db, picoclawLedgerTurn{SessionID: "sess-root", Role: "assistant", Content: "old context answer", TSUnix: 101, ChatID: "telegram:42"})
	seedPicoClawLedgerTurn(t, db, picoclawLedgerTurn{SessionID: "sess-fresh", Role: "user", Content: "fresh context question", TSUnix: 300, ChatID: "telegram:42"})

	root, err := ReadSessionLedger(ctx, db, meta, "sess-root")
	if err != nil {
		t.Fatalf("ReadSessionLedger root: %v", err)
	}
	if len(root.Messages) != 2 || !strings.Contains(root.Messages[0].Content, "old context") {
		t.Fatalf("root history not preserved: %+v", root.Messages)
	}
	if len(root.ResetBoundaries) != 1 || root.ResetBoundaries[0].AfterSessionID != "sess-fresh" {
		t.Fatalf("reset boundaries = %+v, want boundary to sess-fresh", root.ResetBoundaries)
	}
	fresh, err := ReadSessionLedger(ctx, db, meta, "sess-fresh")
	if err != nil {
		t.Fatalf("ReadSessionLedger fresh: %v", err)
	}
	if len(fresh.Messages) != 1 || !strings.Contains(fresh.Messages[0].Content, "fresh context") {
		t.Fatalf("fresh history = %+v, want only fresh generation messages", fresh.Messages)
	}
	resolved, err := meta.ResolveLineageTip(ctx, "sess-root")
	if err != nil {
		t.Fatalf("ResolveLineageTip: %v", err)
	}
	if resolved.LiveSessionID != "sess-fresh" || strings.Join(resolved.Path, " -> ") != "sess-root -> sess-fresh" {
		t.Fatalf("lineage tip = %+v, want non-destructive fresh child", resolved)
	}
}

func TestPicoClawSessionLedger_DurableAttachmentRefsSurviveReopen(t *testing.T) {
	db := openPicoClawLedgerTestDB(t)
	metaJSON := attachmentsMeta(t, []map[string]any{
		{"kind": "document", "file_name": "report.pdf", "media_type": "application/pdf", "source_id": "doc-1", "url": "gormes://attachments/sess-attachments/doc-1"},
		{"kind": "image", "file_name": "scratch.png", "media_type": "image/png", "source_id": "tmp-1", "url": "temp://gateway-cache/scratch.png"},
	})
	seedPicoClawLedgerTurn(t, db, picoclawLedgerTurn{SessionID: "sess-attachments", Role: "user", Content: "inspect attachments", TSUnix: 100, ChatID: "telegram:42", MetaJSON: metaJSON})

	first, err := ReadSessionLedger(context.Background(), db, nil, "sess-attachments")
	if err != nil {
		t.Fatalf("ReadSessionLedger first: %v", err)
	}
	second, err := ReadSessionLedger(context.Background(), db, nil, "sess-attachments")
	if err != nil {
		t.Fatalf("ReadSessionLedger second: %v", err)
	}
	for _, ledger := range []SessionLedger{first, second} {
		if len(ledger.Messages) != 1 || len(ledger.Messages[0].Attachments) != 2 {
			t.Fatalf("attachments = %+v, want two attachment refs", ledger.Messages)
		}
		durable := ledger.Messages[0].Attachments[0]
		if !durable.Durable || durable.URL != "gormes://attachments/sess-attachments/doc-1" || durable.FileName != "report.pdf" {
			t.Fatalf("durable attachment ref = %+v", durable)
		}
		redacted := ledger.Messages[0].Attachments[1]
		if redacted.Durable || redacted.URL != "" || redacted.Evidence != "redacted_non_durable_attachment_ref" {
			t.Fatalf("redacted attachment ref = %+v", redacted)
		}
		if strings.Contains(fmt.Sprintf("%+v", redacted), "gateway-cache") {
			t.Fatalf("redacted attachment leaked temp ref: %+v", redacted)
		}
	}
}

type picoclawLedgerTurn struct {
	SessionID string
	Role      string
	Content   string
	TSUnix    int64
	ChatID    string
	MetaJSON  string
}

func openPicoClawLedgerTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := openSessionDirectoryTestDB(t)
	return db
}

func seedPicoClawLedgerTurn(t *testing.T, db *sql.DB, row picoclawLedgerTurn) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO turns(session_id, role, content, ts_unix, chat_id, meta_json) VALUES (?, ?, ?, ?, ?, ?)`,
		row.SessionID, row.Role, row.Content, row.TSUnix, row.ChatID, nullLedgerString(row.MetaJSON),
	); err != nil {
		t.Fatalf("seed turn %s: %v", row.SessionID, err)
	}
}

func senderMeta(t *testing.T, id, name string) string {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"sender": map[string]any{"id": id, "name": name}})
	if err != nil {
		t.Fatalf("marshal sender meta: %v", err)
	}
	return string(raw)
}

func attachmentsMeta(t *testing.T, attachments []map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"attachments": attachments})
	if err != nil {
		t.Fatalf("marshal attachment meta: %v", err)
	}
	return string(raw)
}

func nullLedgerString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
