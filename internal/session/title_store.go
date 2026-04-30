package session

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	bolt "go.etcd.io/bbolt"
)

// metadataReader is the read side consumed by MetadataTitleStore.
type metadataReader interface {
	GetMetadata(ctx context.Context, sessionID string) (Metadata, bool, error)
}

// autoTitleWriter is the write side used by SetTitle. It performs a
// read-modify-write that unconditionally clears TitleManuallySet, bypassing
// the sticky-true guard in mergeMetadata. This satisfies the auto_title.go
// interface contract: "auto-titles are non-manual; SetTitle clears the manual
// flag atomically with the title write."
type autoTitleWriter interface {
	setTitleAuto(ctx context.Context, sessionID, title string) error
}

// metadataMap combines both sides.
type metadataMap interface {
	metadataReader
	autoTitleWriter
}

// MetadataTitleStore is the concrete SessionTitleStore adapter that wraps a
// session.Map and surfaces TitleManuallySet to PerformAutoTitle. The ctx
// passed to NewMetadataTitleStore is stored on the struct and used for all
// Map I/O so the interface methods Title and SetTitle remain ctx-free,
// matching the narrow SessionTitleStore contract.
type MetadataTitleStore struct {
	ctx context.Context //nolint:containedctx // narrow adapter; ctx is provided at construction
	m   metadataMap
}

// NewMetadataTitleStore constructs an adapter over any *MemMap or *BoltMap.
// ctx is held for Map I/O calls.
func NewMetadataTitleStore(ctx context.Context, m metadataMap) *MetadataTitleStore {
	return &MetadataTitleStore{ctx: ctx, m: m}
}

// metadataReadOnlyMap wraps a bare metadataReader (no autoTitleWriter) for
// tests that only need the read path (Title/error propagation fixtures).
type metadataReadOnlyMap struct {
	metadataReader
}

func (metadataReadOnlyMap) setTitleAuto(_ context.Context, _, _ string) error {
	return nil
}

// NewMetadataTitleStoreFromReader constructs an adapter from a read-only
// source. SetTitle is a no-op on this variant; used only by read-error tests.
func NewMetadataTitleStoreFromReader(ctx context.Context, r metadataReader) *MetadataTitleStore {
	return &MetadataTitleStore{ctx: ctx, m: metadataReadOnlyMap{r}}
}

// Title implements SessionTitleStore. It returns (current, manual, ok, err)
// from the persisted Metadata.
func (s *MetadataTitleStore) Title(sessionID string) (current string, manual bool, ok bool, err error) {
	meta, ok, err := s.m.GetMetadata(s.ctx, sessionID)
	if err != nil {
		return "", false, false, err
	}
	if !ok {
		return "", false, false, nil
	}
	return meta.Title, meta.TitleManuallySet, true, nil
}

// SetTitle implements SessionTitleStore. It persists title for sessionID and
// atomically clears TitleManuallySet so a follow-up Title call returns
// manual=false. Uses the autoTitleWriter path which bypasses mergeMetadata's
// sticky-true guard.
func (s *MetadataTitleStore) SetTitle(sessionID, title string) error {
	return s.m.setTitleAuto(s.ctx, sessionID, title)
}

// setTitleAuto on MemMap performs a locked read-modify-write that directly
// sets Title and clears TitleManuallySet, bypassing mergeMetadata's
// sticky-true guard.
func (m *MemMap) setTitleAuto(ctx context.Context, sessionID, title string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	meta := m.meta[sessionID]
	meta.SessionID = sessionID
	meta.Title = title
	meta.TitleManuallySet = false // explicit clear — auto-titles are non-manual
	meta.UpdatedAt = time.Now().Unix()
	if meta.CreatedAt == 0 {
		meta.CreatedAt = meta.UpdatedAt
	}
	m.meta[sessionID] = meta
	return nil
}

// setTitleAuto on BoltMap performs an atomic bolt transaction that reads the
// existing row, sets Title, and clears TitleManuallySet, bypassing
// mergeMetadata's sticky-true guard.
func (m *BoltMap) setTitleAuto(ctx context.Context, sessionID, title string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.closeMu.Lock()
	db := m.db
	m.closeMu.Unlock()
	if db == nil {
		return errors.New("session: BoltMap is closed")
	}

	return db.Update(func(tx *bolt.Tx) error {
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
		meta.Title = title
		meta.TitleManuallySet = false // explicit clear — auto-titles are non-manual
		meta.UpdatedAt = time.Now().Unix()
		if meta.CreatedAt == 0 {
			meta.CreatedAt = meta.UpdatedAt
		}
		raw, err := json.Marshal(meta)
		if err != nil {
			return err
		}
		return b.Put([]byte(sessionID), raw)
	})
}
