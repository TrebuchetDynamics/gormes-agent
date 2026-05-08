package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

const (
	metadataBucketName = "session_meta_v1"
	chatUserBucketName = "session_chat_users_v1"
)

// ErrUserBindingConflict reports an attempt to bind one canonical chat to
// multiple distinct user IDs.
var ErrUserBindingConflict = errors.New("session: chat already bound to different user_id")

// ErrSessionNotFound reports a missing session lookup in the SQLite
// transcript-backed session directory.
var ErrSessionNotFound = errors.New("session: not found")

// ErrSessionPrefixAmbiguous reports a prefix that matches multiple sessions.
var ErrSessionPrefixAmbiguous = errors.New("session: prefix ambiguous")

// DirectoryEntry is the read model used by CLI session ergonomics. It is
// derived from the native turns table and optional session metadata.
type DirectoryEntry struct {
	ID           string
	Title        string
	Preview      string
	Source       string
	StartedAt    int64
	LastActiveAt int64
	MessageCount int
}

// DirectoryFilter narrows transcript-backed session directory queries.
type DirectoryFilter struct {
	Source string
	Limit  int
}

// Metadata is the first durable identity layer above the raw session map.
// SessionID remains the resume handle; Source+ChatID identify the transport
// chat; UserID is the canonical participant identity that can span chats.
type Metadata struct {
	SessionID                    string               `json:"session_id"`
	Source                       string               `json:"source,omitempty"`
	ChatID                       string               `json:"chat_id,omitempty"`
	UserID                       string               `json:"user_id,omitempty"`
	Title                        string               `json:"title,omitempty"`
	ParentSessionID              string               `json:"parent_session_id,omitempty"`
	LineageKind                  string               `json:"lineage_kind"`
	CreatedAt                    int64                `json:"created_at,omitempty"`
	UpdatedAt                    int64                `json:"updated_at"`
	ResumePending                bool                 `json:"resume_pending,omitempty"`
	ResumeReason                 string               `json:"resume_reason,omitempty"`
	ResumeMarkedAt               int64                `json:"resume_marked_at,omitempty"`
	NonResumableReason           string               `json:"non_resumable_reason,omitempty"`
	NonResumableAt               int64                `json:"non_resumable_at,omitempty"`
	ExpiryFinalized              bool                 `json:"expiry_finalized,omitempty"`
	ExpiryFinalizeStatus         ExpiryFinalizeStatus `json:"expiry_finalize_status,omitempty"`
	ExpiryFinalizeAttempts       int                  `json:"expiry_finalize_attempts,omitempty"`
	ExpiryFinalizeLastError      string               `json:"expiry_finalize_last_error,omitempty"`
	ExpiryFinalizeLastEvidenceAt int64                `json:"expiry_finalize_last_evidence_at,omitempty"`
	MigratedMemoryFlushed        bool                 `json:"migrated_memory_flushed,omitempty"`
	// TokensInTotal and TokensOutTotal persist provider usage totals observed by
	// the gateway for this session so /status can report durable Hermes-style
	// token accounting after the live render frame has gone idle or restarted.
	TokensInTotal  int `json:"tokens_in_total,omitempty"`
	TokensOutTotal int `json:"tokens_out_total,omitempty"`
	// TitleManuallySet is true when the title was set by an operator via
	// /title rather than generated automatically. mergeMetadata treats this
	// field as sticky-true: once set it cannot be cleared by a plain
	// PutMetadata call. Use MetadataTitleStore.SetTitle to clear it atomically
	// (that path uses a read-modify-write that bypasses the sticky guard).
	TitleManuallySet bool       `json:"title_manually_set,omitempty"`
	Goal             *GoalState `json:"goal,omitempty"`
}

// ExpiryFinalizeStatus is persisted evidence for gateway session-expiry
// finalization attempts. Values intentionally match the operator status text.
type ExpiryFinalizeStatus string

const (
	ExpiryFinalizeStatusPending   ExpiryFinalizeStatus = "expiry_finalize_pending"
	ExpiryFinalizeStatusFailed    ExpiryFinalizeStatus = "expiry_finalize_failed"
	ExpiryFinalizeStatusGaveUp    ExpiryFinalizeStatus = "expiry_finalize_gave_up"
	ExpiryFinalizeStatusFinalized ExpiryFinalizeStatus = "expiry_finalized"
)

func normalizeMetadata(meta Metadata) Metadata {
	meta.SessionID = strings.TrimSpace(meta.SessionID)
	meta.Source = strings.TrimSpace(meta.Source)
	meta.ChatID = strings.TrimSpace(meta.ChatID)
	meta.UserID = strings.TrimSpace(meta.UserID)
	meta.Title = strings.TrimSpace(meta.Title)
	meta.ParentSessionID = strings.TrimSpace(meta.ParentSessionID)
	meta.LineageKind = strings.ToLower(strings.TrimSpace(meta.LineageKind))
	meta.ResumeReason = strings.ToLower(strings.TrimSpace(meta.ResumeReason))
	meta.NonResumableReason = strings.ToLower(strings.TrimSpace(meta.NonResumableReason))
	meta.ExpiryFinalizeStatus = ExpiryFinalizeStatus(strings.ToLower(strings.TrimSpace(string(meta.ExpiryFinalizeStatus))))
	meta.ExpiryFinalizeLastError = strings.TrimSpace(meta.ExpiryFinalizeLastError)
	if meta.ExpiryFinalizeAttempts < 0 {
		meta.ExpiryFinalizeAttempts = 0
	}
	if meta.TokensInTotal < 0 {
		meta.TokensInTotal = 0
	}
	if meta.TokensOutTotal < 0 {
		meta.TokensOutTotal = 0
	}
	meta.Goal = CloneGoalState(meta.Goal)
	return meta
}

func mergeMetadata(existing, incoming Metadata) (Metadata, error) {
	out := existing
	out.SessionID = incoming.SessionID
	if incoming.Source != "" {
		out.Source = incoming.Source
	}
	if incoming.ChatID != "" {
		out.ChatID = incoming.ChatID
	}
	if incoming.UserID != "" {
		out.UserID = incoming.UserID
	}
	if incoming.Title != "" {
		out.Title = incoming.Title
	}
	// TitleManuallySet is sticky-true: incoming=true always sets the flag;
	// incoming=false (the default zero value) never clears an existing true.
	// Atomic clearing by auto-title is performed by MetadataTitleStore.SetTitle
	// via a read-modify-write that directly overwrites the stored value.
	if incoming.TitleManuallySet {
		out.TitleManuallySet = true
	}
	if incoming.ParentSessionID != "" {
		if out.ParentSessionID != "" && out.ParentSessionID != incoming.ParentSessionID {
			return Metadata{}, fmt.Errorf("%w: %s parent_session_id already %s", ErrLineageConflict, incoming.SessionID, out.ParentSessionID)
		}
		out.ParentSessionID = incoming.ParentSessionID
	}
	if incoming.LineageKind != "" {
		if out.LineageKind != "" && out.LineageKind != LineageKindPrimary && out.LineageKind != incoming.LineageKind {
			return Metadata{}, fmt.Errorf("%w: %s lineage_kind already %s", ErrLineageConflict, incoming.SessionID, out.LineageKind)
		}
		out.LineageKind = incoming.LineageKind
	}
	if incoming.CreatedAt != 0 && out.CreatedAt == 0 {
		out.CreatedAt = incoming.CreatedAt
	}
	if incoming.UpdatedAt != 0 {
		out.UpdatedAt = incoming.UpdatedAt
	}
	if incoming.ResumePending {
		out.ResumePending = true
	}
	if incoming.ResumeReason != "" {
		out.ResumeReason = incoming.ResumeReason
	}
	if incoming.ResumeMarkedAt != 0 {
		out.ResumeMarkedAt = incoming.ResumeMarkedAt
	}
	if incoming.NonResumableReason != "" {
		out.NonResumableReason = incoming.NonResumableReason
	}
	if incoming.NonResumableAt != 0 {
		out.NonResumableAt = incoming.NonResumableAt
	}
	if incoming.ExpiryFinalized {
		out.ExpiryFinalized = true
	}
	if incoming.ExpiryFinalizeStatus != "" ||
		incoming.ExpiryFinalizeAttempts != 0 ||
		incoming.ExpiryFinalizeLastError != "" ||
		incoming.ExpiryFinalizeLastEvidenceAt != 0 {
		out.ExpiryFinalizeStatus = incoming.ExpiryFinalizeStatus
		out.ExpiryFinalizeAttempts = incoming.ExpiryFinalizeAttempts
		out.ExpiryFinalizeLastError = incoming.ExpiryFinalizeLastError
		out.ExpiryFinalizeLastEvidenceAt = incoming.ExpiryFinalizeLastEvidenceAt
	}
	if incoming.MigratedMemoryFlushed {
		out.MigratedMemoryFlushed = true
	}
	if incoming.TokensInTotal > out.TokensInTotal {
		out.TokensInTotal = incoming.TokensInTotal
	}
	if incoming.TokensOutTotal > out.TokensOutTotal {
		out.TokensOutTotal = incoming.TokensOutTotal
	}
	if incoming.Goal != nil {
		out.Goal = CloneGoalState(incoming.Goal)
	}
	return finalizeMetadata(out), nil
}

func chatBindingKey(source, chatID string) string {
	return source + "\x00" + chatID
}

func sortMetadata(items []Metadata) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt != items[j].UpdatedAt {
			return items[i].UpdatedAt > items[j].UpdatedAt
		}
		return items[i].SessionID < items[j].SessionID
	})
}

func decodeMetadata(raw []byte) (Metadata, error) {
	var meta Metadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		return Metadata{}, err
	}
	applyLegacyMemoryFlushedMigration(raw, &meta)
	return finalizeMetadata(meta), nil
}

func applyLegacyMemoryFlushedMigration(raw []byte, meta *Metadata) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return
	}
	legacyRaw, ok := fields["memory_flushed"]
	if !ok {
		return
	}
	var legacyFlushed bool
	if err := json.Unmarshal(legacyRaw, &legacyFlushed); err != nil || !legacyFlushed {
		return
	}
	meta.MigratedMemoryFlushed = true
	meta.ExpiryFinalized = true
}

func (m *BoltMap) PutMetadata(ctx context.Context, meta Metadata) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.closeMu.Lock()
	db := m.db
	m.closeMu.Unlock()
	if db == nil {
		return errors.New("session: BoltMap is closed")
	}

	meta = normalizeMetadata(meta)
	if err := validateMetadataIdentity(meta); err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		mb := tx.Bucket([]byte(metadataBucketName))
		cb := tx.Bucket([]byte(chatUserBucketName))
		if mb == nil || cb == nil {
			return errors.New("session: metadata buckets missing")
		}

		if raw := mb.Get([]byte(meta.SessionID)); raw != nil {
			existing, err := decodeMetadata(raw)
			if err != nil {
				return fmt.Errorf("session: decode metadata for %q: %w", meta.SessionID, err)
			}
			meta, err = mergeMetadata(existing, meta)
			if err != nil {
				return err
			}
		}
		meta = finalizeMetadata(meta)
		if err := validateMetadata(meta); err != nil {
			return err
		}
		if err := detectLineageLoop(meta.SessionID, meta.ParentSessionID, func(id string) (Metadata, bool, error) {
			raw := mb.Get([]byte(id))
			if raw == nil {
				return Metadata{}, false, nil
			}
			meta, err := decodeMetadata(raw)
			if err != nil {
				return Metadata{}, false, fmt.Errorf("session: decode lineage parent %q: %w", id, err)
			}
			return meta, true, nil
		}); err != nil {
			return err
		}

		if meta.Source != "" && meta.ChatID != "" {
			key := chatBindingKey(meta.Source, meta.ChatID)
			if raw := cb.Get([]byte(key)); len(raw) > 0 {
				bound := strings.TrimSpace(string(raw))
				if meta.UserID == "" {
					meta.UserID = bound
				} else if bound != meta.UserID {
					return fmt.Errorf("%w: %s/%s bound to %s", ErrUserBindingConflict, meta.Source, meta.ChatID, bound)
				}
			}
			if meta.UserID != "" {
				if err := cb.Put([]byte(key), []byte(meta.UserID)); err != nil {
					return fmt.Errorf("session: persist chat binding: %w", err)
				}
			}
		}

		if meta.UpdatedAt == 0 {
			meta.UpdatedAt = time.Now().Unix()
		}
		if meta.CreatedAt == 0 {
			meta.CreatedAt = meta.UpdatedAt
		}
		raw, err := json.Marshal(meta)
		if err != nil {
			return fmt.Errorf("session: encode metadata for %q: %w", meta.SessionID, err)
		}
		return mb.Put([]byte(meta.SessionID), raw)
	})
}

func (m *BoltMap) GetMetadata(ctx context.Context, sessionID string) (Metadata, bool, error) {
	if err := ctx.Err(); err != nil {
		return Metadata{}, false, err
	}
	m.closeMu.Lock()
	db := m.db
	m.closeMu.Unlock()
	if db == nil {
		return Metadata{}, false, errors.New("session: BoltMap is closed")
	}

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return Metadata{}, false, nil
	}

	var (
		meta Metadata
		ok   bool
	)
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(metadataBucketName))
		if b == nil {
			return errors.New("session: metadata bucket missing")
		}
		raw := b.Get([]byte(sessionID))
		if raw == nil {
			return nil
		}
		decoded, err := decodeMetadata(raw)
		if err != nil {
			return fmt.Errorf("session: decode metadata for %q: %w", sessionID, err)
		}
		meta = decoded
		ok = true
		return nil
	})
	if err != nil {
		return Metadata{}, false, err
	}
	return meta, ok, nil
}

func (m *BoltMap) ClearResumePending(ctx context.Context, sessionID string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	m.closeMu.Lock()
	db := m.db
	m.closeMu.Unlock()
	if db == nil {
		return false, errors.New("session: BoltMap is closed")
	}

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return false, nil
	}

	var cleared bool
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(metadataBucketName))
		if b == nil {
			return errors.New("session: metadata bucket missing")
		}
		raw := b.Get([]byte(sessionID))
		if raw == nil {
			return nil
		}
		meta, err := decodeMetadata(raw)
		if err != nil {
			return fmt.Errorf("session: decode metadata for %q: %w", sessionID, err)
		}
		if !meta.ResumePending && meta.ResumeReason == "" && meta.ResumeMarkedAt == 0 {
			return nil
		}
		meta.ResumePending = false
		meta.ResumeReason = ""
		meta.ResumeMarkedAt = 0
		encoded, err := json.Marshal(meta)
		if err != nil {
			return fmt.Errorf("session: encode metadata for %q: %w", sessionID, err)
		}
		if err := b.Put([]byte(sessionID), encoded); err != nil {
			return err
		}
		cleared = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return cleared, nil
}

func (m *BoltMap) ResolveUserID(ctx context.Context, source, chatID string) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	m.closeMu.Lock()
	db := m.db
	m.closeMu.Unlock()
	if db == nil {
		return "", false, errors.New("session: BoltMap is closed")
	}

	source = strings.TrimSpace(source)
	chatID = strings.TrimSpace(chatID)
	if source == "" || chatID == "" {
		return "", false, nil
	}

	var out string
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(chatUserBucketName))
		if b == nil {
			return errors.New("session: chat binding bucket missing")
		}
		raw := b.Get([]byte(chatBindingKey(source, chatID)))
		if raw != nil {
			out = strings.TrimSpace(string(raw))
		}
		return nil
	})
	if err != nil {
		return "", false, err
	}
	return out, out != "", nil
}

func (m *BoltMap) ListMetadataByUserID(ctx context.Context, userID string) ([]Metadata, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.closeMu.Lock()
	db := m.db
	m.closeMu.Unlock()
	if db == nil {
		return nil, errors.New("session: BoltMap is closed")
	}

	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, nil
	}

	var items []Metadata
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(metadataBucketName))
		if b == nil {
			return errors.New("session: metadata bucket missing")
		}
		return b.ForEach(func(_, raw []byte) error {
			meta, err := decodeMetadata(raw)
			if err != nil {
				return fmt.Errorf("session: decode metadata during list: %w", err)
			}
			if meta.UserID == userID {
				items = append(items, meta)
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	sortMetadata(items)
	return items, nil
}

func (m *BoltMap) listAllMetadata(ctx context.Context) ([]Metadata, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.closeMu.Lock()
	db := m.db
	m.closeMu.Unlock()
	if db == nil {
		return nil, errors.New("session: BoltMap is closed")
	}

	var items []Metadata
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(metadataBucketName))
		if b == nil {
			return errors.New("session: metadata bucket missing")
		}
		return b.ForEach(func(_, raw []byte) error {
			meta, err := decodeMetadata(raw)
			if err != nil {
				return fmt.Errorf("session: decode metadata during list: %w", err)
			}
			items = append(items, meta)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	sortMetadata(items)
	return items, nil
}

func (m *BoltMap) ResolveLineageTip(ctx context.Context, sessionID string) (LineageResolution, error) {
	items, err := m.listAllMetadata(ctx)
	if err != nil {
		return LineageResolution{}, err
	}
	return resolveLineageTipFromMetadata(strings.TrimSpace(sessionID), items), nil
}

func (m *MemMap) PutMetadata(ctx context.Context, meta Metadata) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	meta = normalizeMetadata(meta)
	if err := validateMetadataIdentity(meta); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.meta[meta.SessionID]; ok {
		var err error
		meta, err = mergeMetadata(existing, meta)
		if err != nil {
			return err
		}
	}
	meta = finalizeMetadata(meta)
	if err := validateMetadata(meta); err != nil {
		return err
	}
	if err := detectLineageLoop(meta.SessionID, meta.ParentSessionID, func(id string) (Metadata, bool, error) {
		meta, ok := m.meta[id]
		if !ok {
			return Metadata{}, false, nil
		}
		return finalizeMetadata(meta), true, nil
	}); err != nil {
		return err
	}
	if meta.Source != "" && meta.ChatID != "" {
		key := chatBindingKey(meta.Source, meta.ChatID)
		if bound, ok := m.chatUsers[key]; ok {
			if meta.UserID == "" {
				meta.UserID = bound
			} else if bound != meta.UserID {
				return fmt.Errorf("%w: %s/%s bound to %s", ErrUserBindingConflict, meta.Source, meta.ChatID, bound)
			}
		}
		if meta.UserID != "" {
			m.chatUsers[key] = meta.UserID
		}
	}
	if meta.UpdatedAt == 0 {
		meta.UpdatedAt = time.Now().Unix()
	}
	if meta.CreatedAt == 0 {
		meta.CreatedAt = meta.UpdatedAt
	}
	m.meta[meta.SessionID] = meta
	return nil
}

func (m *MemMap) GetMetadata(ctx context.Context, sessionID string) (Metadata, bool, error) {
	if err := ctx.Err(); err != nil {
		return Metadata{}, false, err
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return Metadata{}, false, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	meta, ok := m.meta[sessionID]
	return meta, ok, nil
}

func (m *MemMap) ClearResumePending(ctx context.Context, sessionID string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return false, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	meta, ok := m.meta[sessionID]
	if !ok {
		return false, nil
	}
	if !meta.ResumePending && meta.ResumeReason == "" && meta.ResumeMarkedAt == 0 {
		return false, nil
	}
	meta.ResumePending = false
	meta.ResumeReason = ""
	meta.ResumeMarkedAt = 0
	m.meta[sessionID] = meta
	return true, nil
}

func (m *MemMap) ResolveUserID(ctx context.Context, source, chatID string) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	source = strings.TrimSpace(source)
	chatID = strings.TrimSpace(chatID)
	if source == "" || chatID == "" {
		return "", false, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	userID, ok := m.chatUsers[chatBindingKey(source, chatID)]
	return userID, ok, nil
}

func (m *MemMap) ListMetadataByUserID(ctx context.Context, userID string) ([]Metadata, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	items := make([]Metadata, 0, len(m.meta))
	for _, meta := range m.meta {
		if meta.UserID == userID {
			items = append(items, meta)
		}
	}
	sortMetadata(items)
	return items, nil
}

func (m *MemMap) listAllMetadata(ctx context.Context) ([]Metadata, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	items := make([]Metadata, 0, len(m.meta))
	for _, meta := range m.meta {
		items = append(items, finalizeMetadata(meta))
	}
	sortMetadata(items)
	return items, nil
}

func (m *MemMap) ResolveLineageTip(ctx context.Context, sessionID string) (LineageResolution, error) {
	items, err := m.listAllMetadata(ctx)
	if err != nil {
		return LineageResolution{}, err
	}
	return resolveLineageTipFromMetadata(strings.TrimSpace(sessionID), items), nil
}

// ListAllMetadata returns all session metadata entries, sorted by most recent first.
func (m *BoltMap) ListAllMetadata(ctx context.Context) ([]Metadata, error) {
	return m.listAllMetadata(ctx)
}

// ListAllMetadata returns all session metadata entries, sorted by most recent first.
func (m *MemMap) ListAllMetadata(ctx context.Context) ([]Metadata, error) {
	return m.listAllMetadata(ctx)
}

// ListDirectorySessions lists sessions from the native turns table in MRU
// order. Last activity is MAX(ts_unix); legacy single-turn rows naturally fall
// back to started_at because MIN and MAX are equal.
func ListDirectorySessions(ctx context.Context, db *sql.DB, filter DirectoryFilter) ([]DirectoryEntry, error) {
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

	byID := make(map[string]*DirectoryEntry)
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
			entry = &DirectoryEntry{
				ID:           id,
				StartedAt:    ts,
				LastActiveAt: ts,
				Source:       sourceFromDirectoryChatID(chatID),
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
			entry.Source = sourceFromDirectoryChatID(chatID)
		}
		entry.MessageCount++
		if entry.Preview == "" && strings.TrimSpace(role) == "user" {
			entry.Preview = strings.TrimSpace(content)
		}
		if entry.Preview == "" {
			entry.Preview = strings.TrimSpace(content)
		}
		if entry.Title == "" {
			entry.Title = titleFromDirectoryMeta(metaJSON)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("session: iterate directory turns: %w", err)
	}

	source := strings.ToLower(strings.TrimSpace(filter.Source))
	out := make([]DirectoryEntry, 0, len(byID))
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

// ResolveMostRecentSession returns the most recently active session for source.
// Empty stores return an empty id and nil error to match Hermes continue logic.
func ResolveMostRecentSession(ctx context.Context, db *sql.DB, source string) (string, error) {
	sessions, err := ListDirectorySessions(ctx, db, DirectoryFilter{Source: source, Limit: 1})
	if err != nil {
		return "", err
	}
	if len(sessions) == 0 {
		return "", nil
	}
	return sessions[0].ID, nil
}

// ResolveSessionIDPrefix resolves exact ids or unique prefixes.
func ResolveSessionIDPrefix(ctx context.Context, db *sql.DB, prefix string) (string, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return "", ErrSessionNotFound
	}
	sessions, err := ListDirectorySessions(ctx, db, DirectoryFilter{})
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

// DeleteDirectorySession deletes all turns for a resolved session id.
func DeleteDirectorySession(ctx context.Context, db *sql.DB, sessionID string) (bool, error) {
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

// PruneDirectorySessions deletes sessions whose last activity is older than
// cutoffUnix. It returns the number of session ids removed, not turn rows.
func PruneDirectorySessions(ctx context.Context, db *sql.DB, cutoffUnix int64, source string) (int, error) {
	sessions, err := ListDirectorySessions(ctx, db, DirectoryFilter{Source: source})
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
		if _, err := DeleteDirectorySession(ctx, db, id); err != nil {
			return 0, err
		}
	}
	return len(ids), nil
}

func sourceFromDirectoryChatID(chatID string) string {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return "cli"
	}
	if before, _, ok := strings.Cut(chatID, ":"); ok && strings.TrimSpace(before) != "" {
		return strings.ToLower(strings.TrimSpace(before))
	}
	return "cli"
}

func titleFromDirectoryMeta(raw string) string {
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
