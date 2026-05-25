package session

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	bolt "go.etcd.io/bbolt"
)

// SetLabels persists the operator labels/bookmarks associated with a session.
// It preserves all other metadata fields and allows callers to clear labels by
// passing nil or an empty slice.
func (m *MemMap) SetLabels(ctx context.Context, sessionID string, labels []string, now time.Time) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	sessionID = normalizeMetadata(Metadata{SessionID: sessionID}).SessionID
	if sessionID == "" {
		return nil, errors.New("session: labels session_id is required")
	}
	clean := normalizeLabels(labels)
	m.mu.Lock()
	defer m.mu.Unlock()
	meta := m.meta[sessionID]
	meta.SessionID = sessionID
	meta.Labels = append([]string(nil), clean...)
	if !now.IsZero() {
		meta.UpdatedAt = now.Unix()
	} else if meta.UpdatedAt == 0 {
		meta.UpdatedAt = time.Now().Unix()
	}
	if meta.CreatedAt == 0 {
		meta.CreatedAt = meta.UpdatedAt
	}
	m.meta[sessionID] = finalizeMetadata(meta)
	return append([]string(nil), clean...), nil
}

// SetLabels persists the operator labels/bookmarks associated with a session.
// It preserves all other metadata fields and allows callers to clear labels by
// passing nil or an empty slice.
func (m *BoltMap) SetLabels(ctx context.Context, sessionID string, labels []string, now time.Time) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	sessionID = normalizeMetadata(Metadata{SessionID: sessionID}).SessionID
	if sessionID == "" {
		return nil, errors.New("session: labels session_id is required")
	}
	m.closeMu.Lock()
	db := m.db
	m.closeMu.Unlock()
	if db == nil {
		return nil, errors.New("session: BoltMap is closed")
	}
	clean := normalizeLabels(labels)
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(metadataBucketName))
		if b == nil {
			return errors.New("session: metadata bucket missing")
		}
		var meta Metadata
		if raw := b.Get([]byte(sessionID)); raw != nil {
			var err error
			meta, err = decodeMetadata(raw)
			if err != nil {
				return err
			}
		}
		meta.SessionID = sessionID
		meta.Labels = append([]string(nil), clean...)
		if !now.IsZero() {
			meta.UpdatedAt = now.Unix()
		} else if meta.UpdatedAt == 0 {
			meta.UpdatedAt = time.Now().Unix()
		}
		if meta.CreatedAt == 0 {
			meta.CreatedAt = meta.UpdatedAt
		}
		raw, err := json.Marshal(finalizeMetadata(meta))
		if err != nil {
			return err
		}
		return b.Put([]byte(sessionID), raw)
	})
	if err != nil {
		return nil, err
	}
	return append([]string(nil), clean...), nil
}
