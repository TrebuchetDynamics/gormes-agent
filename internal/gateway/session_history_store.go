package gateway

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/transcript"
)

var errSessionHistoryNotFound = errors.New("gateway: session history not found")

type sqlSessionHistoryStore struct {
	db *sql.DB
}

type sqlHistoryTurn struct {
	ID        int64
	Role      string
	Content   string
	TSUnix    int64
	ChatID    string
	MetaJSON  string
	TurnKey   string
	SyncState string
	SyncWhy   string
}

func NewSQLSessionHistoryStore(db *sql.DB) SessionHistoryStore {
	if db == nil {
		return nil
	}
	return &sqlSessionHistoryStore{db: db}
}

func (s *sqlSessionHistoryStore) LoadSessionHistory(ctx context.Context, sessionID string) ([]llm.Message, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, errSessionHistoryNotFound
	}
	messages, err := transcript.LoadMessages(ctx, s.db, sessionID)
	if err != nil {
		if errors.Is(err, transcript.ErrSessionNotFound) {
			return nil, errSessionHistoryNotFound
		}
		return nil, err
	}
	out := make([]llm.Message, 0, len(messages))
	for _, msg := range messages {
		out = append(out, llm.Message{Role: msg.Role, Content: msg.Content})
	}
	return out, nil
}

func (s *sqlSessionHistoryStore) RewriteSessionHistory(ctx context.Context, sessionID string, history []llm.Message) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return errSessionHistoryNotFound
	}
	turns, err := s.loadTurns(ctx, sessionID)
	if err != nil {
		return err
	}
	if len(turns) == 0 {
		return errSessionHistoryNotFound
	}
	keep := commonHistoryPrefix(turns, history)
	if keep == len(history) {
		return s.deleteAfterOrdinal(ctx, sessionID, keep)
	}
	return s.replaceHistory(ctx, sessionID, history)
}

func (s *sqlSessionHistoryStore) RewindSessionHistory(ctx context.Context, sessionID string, userTurns int) (SessionHistoryRewindResult, error) {
	if userTurns <= 0 {
		userTurns = 1
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return SessionHistoryRewindResult{}, errSessionHistoryNotFound
	}
	turns, err := s.loadTurns(ctx, sessionID)
	if err != nil {
		return SessionHistoryRewindResult{}, err
	}
	idx := nthUserTurnIndexFromEnd(turns, userTurns)
	if idx < 0 {
		return SessionHistoryRewindResult{}, errSessionHistoryNotFound
	}
	if err := s.deleteAfterOrdinal(ctx, sessionID, idx); err != nil {
		return SessionHistoryRewindResult{}, err
	}
	kept := make([]llm.Message, 0, idx)
	for _, turn := range turns[:idx] {
		kept = append(kept, llm.Message{Role: turn.Role, Content: turn.Content})
	}
	return SessionHistoryRewindResult{
		SessionID:    sessionID,
		History:      kept,
		TargetText:   turns[idx].Content,
		TurnsUndone:  userTurns,
		RewoundCount: len(turns) - idx,
	}, nil
}

func (s *sqlSessionHistoryStore) loadTurns(ctx context.Context, sessionID string) ([]sqlHistoryTurn, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, role, content, ts_unix, COALESCE(chat_id, ''), COALESCE(meta_json, ''), COALESCE(turn_key, ''), COALESCE(memory_sync_status, 'ready'), COALESCE(memory_sync_reason, '')
		FROM turns
		WHERE session_id = ?
		ORDER BY ts_unix ASC, id ASC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("gateway session history: query %q: %w", sessionID, err)
	}
	defer rows.Close()
	var turns []sqlHistoryTurn
	for rows.Next() {
		var turn sqlHistoryTurn
		if err := rows.Scan(&turn.ID, &turn.Role, &turn.Content, &turn.TSUnix, &turn.ChatID, &turn.MetaJSON, &turn.TurnKey, &turn.SyncState, &turn.SyncWhy); err != nil {
			return nil, fmt.Errorf("gateway session history: scan %q: %w", sessionID, err)
		}
		turns = append(turns, turn)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("gateway session history: iterate %q: %w", sessionID, err)
	}
	return turns, nil
}

func (s *sqlSessionHistoryStore) deleteAfterOrdinal(ctx context.Context, sessionID string, keep int) error {
	if keep < 0 {
		keep = 0
	}
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM turns
		WHERE session_id = ?
		  AND id IN (
			SELECT id FROM turns
			WHERE session_id = ?
			ORDER BY ts_unix ASC, id ASC
			LIMIT -1 OFFSET ?
		  )
	`, sessionID, sessionID, keep)
	if err != nil {
		return fmt.Errorf("gateway session history: delete rewound turns: %w", err)
	}
	return nil
}

func (s *sqlSessionHistoryStore) replaceHistory(ctx context.Context, sessionID string, history []llm.Message) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("gateway session history: begin rewrite: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM turns WHERE session_id = ?`, sessionID); err != nil {
		return fmt.Errorf("gateway session history: clear rewrite: %w", err)
	}
	now := time.Now().Unix()
	for i, msg := range history {
		role := strings.TrimSpace(msg.Role)
		if role != "user" && role != "assistant" {
			continue
		}
		if strings.TrimSpace(msg.Content) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO turns(session_id, role, content, ts_unix, memory_sync_status)
			VALUES(?, ?, ?, ?, 'ready')
		`, sessionID, role, msg.Content, now+int64(i)); err != nil {
			return fmt.Errorf("gateway session history: insert rewrite: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("gateway session history: commit rewrite: %w", err)
	}
	return nil
}

func commonHistoryPrefix(turns []sqlHistoryTurn, history []llm.Message) int {
	limit := len(turns)
	if len(history) < limit {
		limit = len(history)
	}
	idx := 0
	for idx < limit {
		if strings.TrimSpace(turns[idx].Role) != strings.TrimSpace(history[idx].Role) || turns[idx].Content != history[idx].Content {
			break
		}
		idx++
	}
	return idx
}

func nthUserTurnIndexFromEnd(turns []sqlHistoryTurn, n int) int {
	if n <= 0 {
		n = 1
	}
	seen := 0
	for i := len(turns) - 1; i >= 0; i-- {
		if strings.TrimSpace(turns[i].Role) != "user" {
			continue
		}
		seen++
		if seen == n {
			return i
		}
	}
	return -1
}
