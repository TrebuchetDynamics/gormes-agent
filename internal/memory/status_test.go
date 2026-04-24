package memory

import (
	"context"
	"testing"
	"time"
)

func TestReadExtractorStatus_SummarizesQueueAndDeadLetters(t *testing.T) {
	store, err := OpenSqlite(t.TempDir()+"/memory.db", 8, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(context.Background())

	now := time.Date(2026, 4, 22, 15, 4, 5, 0, time.UTC).Unix()
	_, err = store.DB().Exec(
		`INSERT INTO turns(session_id, role, content, ts_unix, chat_id, extracted, extraction_attempts, extraction_error, cron)
		 VALUES
		 ('sess-1', 'user', 'queued turn', ?, 'telegram:1', 0, 0, NULL, 0),
		 ('sess-2', 'user', 'dead letter one', ?, 'telegram:2', 2, 3, 'malformed JSON', 0),
		 ('sess-3', 'assistant', 'dead letter two', ?, 'discord:9', 2, 4, 'upstream timeout', 0),
		 ('cron-1', 'user', 'cron turn', ?, 'cron:job', 0, 0, NULL, 1)`,
		now, now+1, now+2, now+3,
	)
	if err != nil {
		t.Fatalf("seed turns: %v", err)
	}

	got, err := ReadExtractorStatus(context.Background(), store.DB(), 5)
	if err != nil {
		t.Fatalf("ReadExtractorStatus: %v", err)
	}

	if got.QueueDepth != 1 {
		t.Fatalf("QueueDepth = %d, want 1", got.QueueDepth)
	}
	if got.DeadLetterCount != 2 {
		t.Fatalf("DeadLetterCount = %d, want 2", got.DeadLetterCount)
	}
	if got.WorkerHealth != "degraded" {
		t.Fatalf("WorkerHealth = %q, want %q", got.WorkerHealth, "degraded")
	}
	if len(got.RecentDeadLetters) != 2 {
		t.Fatalf("RecentDeadLetters len = %d, want 2", len(got.RecentDeadLetters))
	}
	if got.RecentDeadLetters[0].SessionID != "sess-3" {
		t.Fatalf("RecentDeadLetters[0].SessionID = %q, want %q", got.RecentDeadLetters[0].SessionID, "sess-3")
	}
	if got.RecentDeadLetters[0].Error != "upstream timeout" {
		t.Fatalf("RecentDeadLetters[0].Error = %q, want %q", got.RecentDeadLetters[0].Error, "upstream timeout")
	}
	if got.RecentDeadLetters[1].SessionID != "sess-2" {
		t.Fatalf("RecentDeadLetters[1].SessionID = %q, want %q", got.RecentDeadLetters[1].SessionID, "sess-2")
	}
}

func TestReadExtractorStatus_BuildsDeterministicDeadLetterErrorSummary(t *testing.T) {
	store, err := OpenSqlite(t.TempDir()+"/memory.db", 8, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(context.Background())

	now := time.Date(2026, 4, 22, 16, 0, 0, 0, time.UTC).Unix()
	_, err = store.DB().Exec(
		`INSERT INTO turns(session_id, role, content, ts_unix, chat_id, extracted, extraction_attempts, extraction_error, cron)
		 VALUES
		 ('sess-1', 'user', 'dead letter one', ?, 'telegram:1', 2, 3, 'malformed JSON', 0),
		 ('sess-2', 'user', 'dead letter two', ?, 'telegram:2', 2, 4, 'upstream timeout', 0),
		 ('sess-3', 'assistant', 'dead letter three', ?, 'discord:9', 2, 5, 'malformed JSON', 0)`,
		now, now+1, now+2,
	)
	if err != nil {
		t.Fatalf("seed turns: %v", err)
	}

	got, err := ReadExtractorStatus(context.Background(), store.DB(), 5)
	if err != nil {
		t.Fatalf("ReadExtractorStatus: %v", err)
	}

	if len(got.ErrorSummary) != 2 {
		t.Fatalf("ErrorSummary len = %d, want 2", len(got.ErrorSummary))
	}
	if got.ErrorSummary[0].Error != "malformed JSON" || got.ErrorSummary[0].Count != 2 {
		t.Fatalf("ErrorSummary[0] = %+v, want malformed JSON x2", got.ErrorSummary[0])
	}
	if got.ErrorSummary[1].Error != "upstream timeout" || got.ErrorSummary[1].Count != 1 {
		t.Fatalf("ErrorSummary[1] = %+v, want upstream timeout x1", got.ErrorSummary[1])
	}
}
