package memory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// DeadLetterSummary is the operator-facing shape for one recent turn that
// exhausted extractor retries and was parked in the dead-letter state.
type DeadLetterSummary struct {
	ID        int64
	SessionID string
	ChatID    string
	Attempts  int
	Error     string
}

// ExtractorStatus is the Phase 3.E.4 read model behind `gormes memory status`.
type ExtractorStatus struct {
	QueueDepth        int
	DeadLetterCount   int
	WorkerHealth      string
	RecentDeadLetters []DeadLetterSummary
}

// ReadExtractorStatus summarizes extractor backlog and recent dead letters from
// the persisted SQLite turns table. The worker is async and ephemeral, so
// health is inferred from durable queue/dead-letter state instead of process
// liveness.
func ReadExtractorStatus(ctx context.Context, db *sql.DB, deadLetterLimit int) (ExtractorStatus, error) {
	if db == nil {
		return ExtractorStatus{}, errors.New("memory: nil db")
	}
	if deadLetterLimit <= 0 {
		deadLetterLimit = 5
	}

	var status ExtractorStatus
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM turns WHERE extracted = 0 AND cron = 0`).Scan(&status.QueueDepth); err != nil {
		return ExtractorStatus{}, fmt.Errorf("memory: queue depth: %w", err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM turns WHERE extracted = 2 AND cron = 0`).Scan(&status.DeadLetterCount); err != nil {
		return ExtractorStatus{}, fmt.Errorf("memory: dead-letter count: %w", err)
	}
	status.WorkerHealth = extractorWorkerHealth(status.QueueDepth, status.DeadLetterCount)

	rows, err := db.QueryContext(ctx,
		`SELECT id, session_id, chat_id, extraction_attempts, COALESCE(extraction_error, '')
		 FROM turns
		 WHERE extracted = 2 AND cron = 0
		 ORDER BY id DESC
		 LIMIT ?`,
		deadLetterLimit,
	)
	if err != nil {
		return ExtractorStatus{}, fmt.Errorf("memory: recent dead letters: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var dl DeadLetterSummary
		if err := rows.Scan(&dl.ID, &dl.SessionID, &dl.ChatID, &dl.Attempts, &dl.Error); err != nil {
			return ExtractorStatus{}, fmt.Errorf("memory: scan dead letter: %w", err)
		}
		status.RecentDeadLetters = append(status.RecentDeadLetters, dl)
	}
	if err := rows.Err(); err != nil {
		return ExtractorStatus{}, fmt.Errorf("memory: recent dead letters rows: %w", err)
	}

	return status, nil
}

func extractorWorkerHealth(queueDepth, deadLetterCount int) string {
	switch {
	case deadLetterCount > 0:
		return "degraded"
	case queueDepth > 0:
		return "backlog"
	default:
		return "idle"
	}
}
