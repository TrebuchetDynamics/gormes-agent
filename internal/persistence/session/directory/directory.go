// Package directory builds the transcript-backed session directory read model.
//
// It owns SQL reads against the native turns table plus pure resolution helpers.
// It exposes small value types and functions for listing, resolving, deleting,
// and pruning sessions. It must never know about BoltMap/MemMap metadata stores,
// TUI rendering, CLI command handling, or transport/channel orchestration.
package directory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session/chatid"
)

// ErrSessionNotFound reports a missing session lookup in the SQLite
// transcript-backed session directory.
var ErrSessionNotFound = errors.New("session: not found")

// ErrSessionPrefixAmbiguous reports a prefix that matches multiple sessions.
var ErrSessionPrefixAmbiguous = errors.New("session: prefix ambiguous")

// Entry is the read model used by CLI session ergonomics. It is derived from
// the native turns table and optional session metadata.
type Entry struct {
	ID           string
	Title        string
	Preview      string
	Source       string
	StartedAt    int64
	LastActiveAt int64
	MessageCount int
}

// Filter narrows transcript-backed session directory queries.
type Filter struct {
	Source string
	Limit  int
}

// List lists sessions from the native turns table in MRU order. Last activity
// is MAX(ts_unix); legacy single-turn rows naturally fall back to started_at
// because MIN and MAX are equal.
func List(ctx context.Context, db *sql.DB, filter Filter) ([]Entry, error) {
	if db == nil {
		return nil, errors.New("session: directory db is nil")
	}
	rows, err := db.QueryContext(ctx, `SELECT session_id, role, content, ts_unix, COALESCE(chat_id, ''), COALESCE(meta_json, '') FROM turns ORDER BY session_id, ts_unix, id`)
	if err != nil {
		// Fresh-install path: the `turns` table is created lazily on
		// the first turn write, so a brand-new memory.db has no table
		// yet. Treat that as the empty state (caller renders "No
		// sessions found.") instead of surfacing a raw SQL error.
		if strings.Contains(err.Error(), "no such table: turns") {
			return nil, nil
		}
		return nil, fmt.Errorf("session: list directory turns: %w", err)
	}
	defer rows.Close()

	byID := make(map[string]*Entry)
	seenTurns := make(map[string]struct{})
	for rows.Next() {
		var id, role, content, chatID, metaJSON string
		var ts int64
		if err := rows.Scan(&id, &role, &content, &ts, &chatID, &metaJSON); err != nil {
			return nil, fmt.Errorf("session: scan directory turn: %w", err)
		}
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		entry := byID[id]
		if entry == nil {
			entry = &Entry{
				ID:           id,
				StartedAt:    ts,
				LastActiveAt: ts,
				Source:       chatid.SourceFromTranscriptChatID(chatID),
			}
			byID[id] = entry
		}
		if ts < entry.StartedAt {
			entry.StartedAt = ts
		}
		if ts > entry.LastActiveAt {
			entry.LastActiveAt = ts
		}
		if entry.Source == "" || entry.Source == "cli" {
			entry.Source = chatid.SourceFromTranscriptChatID(chatID)
		}
		if entry.Title == "" {
			entry.Title = titleFromMeta(metaJSON)
		}
		dedupeKey := turnDedupeKey(id, role, content, ts)
		if _, ok := seenTurns[dedupeKey]; ok {
			continue
		}
		seenTurns[dedupeKey] = struct{}{}
		entry.MessageCount++
		if entry.Preview == "" && strings.TrimSpace(role) == "user" {
			entry.Preview = strings.TrimSpace(content)
		}
		if entry.Preview == "" {
			entry.Preview = strings.TrimSpace(content)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("session: iterate directory turns: %w", err)
	}

	source := strings.ToLower(strings.TrimSpace(filter.Source))
	out := make([]Entry, 0, len(byID))
	for _, entry := range byID {
		if source != "" && strings.ToLower(entry.Source) != source {
			continue
		}
		out = append(out, *entry)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LastActiveAt != out[j].LastActiveAt {
			return out[i].LastActiveAt > out[j].LastActiveAt
		}
		if out[i].StartedAt != out[j].StartedAt {
			return out[i].StartedAt > out[j].StartedAt
		}
		return out[i].ID < out[j].ID
	})
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}

// ResolveMostRecent returns the most recently active session for source. Empty
// stores return an empty id and nil error to match Hermes continue logic.
func ResolveMostRecent(ctx context.Context, db *sql.DB, source string) (string, error) {
	sessions, err := List(ctx, db, Filter{Source: source, Limit: 1})
	if err != nil {
		return "", err
	}
	if len(sessions) == 0 {
		return "", nil
	}
	return sessions[0].ID, nil
}

// ResolvePrefix resolves exact ids or unique prefixes.
func ResolvePrefix(ctx context.Context, db *sql.DB, prefix string) (string, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return "", ErrSessionNotFound
	}
	sessions, err := List(ctx, db, Filter{})
	if err != nil {
		return "", err
	}
	var matches []string
	for _, session := range sessions {
		if session.ID == prefix {
			return session.ID, nil
		}
		if strings.HasPrefix(session.ID, prefix) {
			matches = append(matches, session.ID)
		}
	}
	switch len(matches) {
	case 0:
		return "", ErrSessionNotFound
	case 1:
		return matches[0], nil
	default:
		sort.Strings(matches)
		return "", fmt.Errorf("%w: %s matches %s", ErrSessionPrefixAmbiguous, prefix, strings.Join(matches, ", "))
	}
}

// Delete deletes all turns for a resolved session id.
func Delete(ctx context.Context, db *sql.DB, sessionID string) (bool, error) {
	if db == nil {
		return false, errors.New("session: directory db is nil")
	}
	res, err := db.ExecContext(ctx, `DELETE FROM turns WHERE session_id = ?`, strings.TrimSpace(sessionID))
	if err != nil {
		return false, fmt.Errorf("session: delete %q: %w", sessionID, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("session: delete rows affected: %w", err)
	}
	return n > 0, nil
}

// Prune deletes sessions whose last activity is older than cutoffUnix. It
// returns the number of session ids removed, not turn rows.
func Prune(ctx context.Context, db *sql.DB, cutoffUnix int64, source string) (int, error) {
	sessions, err := List(ctx, db, Filter{Source: source})
	if err != nil {
		return 0, err
	}
	var ids []string
	for _, session := range sessions {
		if session.LastActiveAt < cutoffUnix {
			ids = append(ids, session.ID)
		}
	}
	for _, id := range ids {
		if _, err := Delete(ctx, db, id); err != nil {
			return 0, err
		}
	}
	return len(ids), nil
}

func turnDedupeKey(sessionID, role, content string, ts int64) string {
	return strings.Join([]string{sessionID, strings.TrimSpace(role), fmt.Sprintf("%d", ts), content}, "\x00")
}

func titleFromMeta(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var meta struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return ""
	}
	return strings.TrimSpace(meta.Title)
}
