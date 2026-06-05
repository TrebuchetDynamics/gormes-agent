package goncho

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type GonchoMarkdownStoreConfig struct {
	Path                  string
	DefaultObserverPeerID string
}

type GonchoMarkdownStore struct {
	db     *sql.DB
	Config GonchoMarkdownStoreConfig
}

type GonchoMarkdownReloadResult struct {
	Inserted        int
	Updated         int
	Tombstoned      int
	Conflicts       []GonchoMarkdownConflict
	NetworkRequired bool
	OllamaRequired  bool
}

type GonchoMarkdownExportResult struct {
	Exported        int
	NetworkRequired bool
	OllamaRequired  bool
}

type GonchoMarkdownConflict struct {
	MemoryID string
	Reason   string
}

func NewGonchoMarkdownStore(db *sql.DB, cfg GonchoMarkdownStoreConfig) *GonchoMarkdownStore {
	return &GonchoMarkdownStore{db: db, Config: cfg}
}

func (s *GonchoMarkdownStore) Reload(ctx context.Context) (GonchoMarkdownReloadResult, error) {
	var result GonchoMarkdownReloadResult
	if s == nil || s.db == nil {
		return result, errors.New("memory: nil goncho markdown store")
	}
	body, err := os.ReadFile(s.Config.Path)
	if err != nil {
		return result, fmt.Errorf("memory: read goncho markdown: %w", err)
	}
	doc, err := ParseGonchoMemoryV1Markdown(body)
	if err != nil {
		return result, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("memory: begin goncho markdown reload: %w", err)
	}
	defer tx.Rollback()

	for _, item := range doc.Items {
		item = normalizeGonchoMarkdownReloadItem(item)
		validationItem := item
		validationItem.Checksum = GonchoMemoryV1Checksum(validationItem.Content)
		if err := ValidateGonchoMemoryV1Item(validationItem); err != nil {
			return result, err
		}
		action, err := s.reloadItem(ctx, tx, item)
		if err != nil {
			return result, err
		}
		switch action.reason {
		case "":
			if action.updated {
				result.Updated++
			} else {
				result.Inserted++
			}
			if item.State == gonchoMemoryV1StateTombstoned {
				result.Tombstoned++
			}
		default:
			result.Conflicts = append(result.Conflicts, GonchoMarkdownConflict{
				MemoryID: item.MemoryID,
				Reason:   action.reason,
			})
		}
	}
	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("memory: commit goncho markdown reload: %w", err)
	}
	return result, nil
}

type gonchoMarkdownReloadAction struct {
	updated bool
	reason  string
}

func (s *GonchoMarkdownStore) reloadItem(ctx context.Context, tx *sql.Tx, item GonchoMemoryV1Item) (gonchoMarkdownReloadAction, error) {
	var existingRevision int
	var existingContent string
	err := tx.QueryRowContext(ctx, `
		SELECT revision, content
		FROM goncho_memory_items
		WHERE memory_id = ?
	`, item.MemoryID).Scan(&existingRevision, &existingContent)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return gonchoMarkdownReloadAction{}, fmt.Errorf("memory: read goncho markdown item %s: %w", item.MemoryID, err)
	}
	if err == nil {
		if existingRevision == item.Revision && existingContent != item.Content {
			existingChecksum := GonchoMemoryV1Checksum(existingContent)
			incomingChecksum := strings.TrimSpace(item.Checksum)
			if incomingChecksum == "" || incomingChecksum == existingChecksum {
				item.Revision = existingRevision + 1
				item.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
				item.Checksum = GonchoMemoryV1Checksum(item.Content)
				return gonchoMarkdownReloadAction{updated: true}, s.upsertItem(ctx, tx, item)
			}
			return gonchoMarkdownReloadAction{reason: "same_revision_content_mismatch"}, nil
		}
		if existingRevision > item.Revision {
			return gonchoMarkdownReloadAction{reason: "stale_revision"}, nil
		}
		item.Checksum = GonchoMemoryV1Checksum(item.Content)
		if existingRevision == item.Revision && existingContent == item.Content {
			return gonchoMarkdownReloadAction{updated: true}, s.upsertItem(ctx, tx, item)
		}
		return gonchoMarkdownReloadAction{updated: true}, s.upsertItem(ctx, tx, item)
	}
	item.Checksum = GonchoMemoryV1Checksum(item.Content)
	return gonchoMarkdownReloadAction{}, s.upsertItem(ctx, tx, item)
}

func (s *GonchoMarkdownStore) upsertItem(ctx context.Context, tx *sql.Tx, item GonchoMemoryV1Item) error {
	tags, err := json.Marshal(item.Tags)
	if err != nil {
		return fmt.Errorf("memory: encode goncho tags: %w", err)
	}
	createdAt, err := gonchoMarkdownUnixTime(item.CreatedAt)
	if err != nil {
		return err
	}
	updatedAt, err := gonchoMarkdownUnixTime(item.UpdatedAt)
	if err != nil {
		return err
	}
	tombstonedAt, err := gonchoMarkdownNullableUnixTime(item.TombstonedAt)
	if err != nil {
		return err
	}
	active := 1
	if item.State == gonchoMemoryV1StateTombstoned {
		active = 0
	}
	observer := s.Config.DefaultObserverPeerID
	if observer == "" {
		observer = item.AgentID
	}
	if item.ProvenanceJSON == "" {
		item.ProvenanceJSON = "{}"
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO goncho_memory_items(
			memory_id,
			contract_version,
			agent_id,
			workspace_id,
			observer_peer_id,
			peer_id,
			session_key,
			source_kind,
			content,
			revision,
			active,
			tombstoned_at,
			tombstone_reason,
			scope,
			provenance_json,
			tags_json,
			importance,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(memory_id) DO UPDATE SET
			contract_version = excluded.contract_version,
			agent_id = excluded.agent_id,
			workspace_id = excluded.workspace_id,
			observer_peer_id = excluded.observer_peer_id,
			peer_id = excluded.peer_id,
			session_key = excluded.session_key,
			source_kind = excluded.source_kind,
			content = excluded.content,
			revision = excluded.revision,
			active = excluded.active,
			tombstoned_at = excluded.tombstoned_at,
			tombstone_reason = excluded.tombstone_reason,
			scope = excluded.scope,
			provenance_json = excluded.provenance_json,
			tags_json = excluded.tags_json,
			importance = excluded.importance,
			created_at = excluded.created_at,
			updated_at = excluded.updated_at
	`, item.MemoryID, GonchoMemoryV1ContractVersion, item.AgentID, item.WorkspaceID, observer, item.PeerID, item.SessionID, item.SourceKind, item.Content, item.Revision, active, tombstonedAt, item.TombstoneReason, item.Scope, item.ProvenanceJSON, string(tags), item.Importance, createdAt, updatedAt)
	if err != nil {
		return fmt.Errorf("memory: upsert goncho markdown item %s: %w", item.MemoryID, err)
	}
	return nil
}

func (s *GonchoMarkdownStore) Export(ctx context.Context) (GonchoMarkdownExportResult, error) {
	var result GonchoMarkdownExportResult
	if s == nil || s.db == nil {
		return result, errors.New("memory: nil goncho markdown store")
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT memory_id, agent_id, workspace_id, peer_id, session_key, source_kind,
		       content, revision, active, tombstoned_at, tombstone_reason, scope,
		       provenance_json, tags_json, importance, created_at, updated_at
		FROM goncho_memory_items
		ORDER BY memory_id
	`)
	if err != nil {
		return result, fmt.Errorf("memory: export goncho markdown: %w", err)
	}
	defer rows.Close()

	doc := GonchoMemoryV1Document{
		FormatVersion:   GonchoMemoryV1MarkdownFormat,
		ContractVersion: GonchoMemoryV1ContractVersion,
	}
	for rows.Next() {
		var item GonchoMemoryV1Item
		var active int
		var tombstonedAt sql.NullInt64
		var tombstoneReason sql.NullString
		var tagsRaw string
		var createdAt, updatedAt int64
		if err := rows.Scan(&item.MemoryID, &item.AgentID, &item.WorkspaceID, &item.PeerID, &item.SessionID, &item.SourceKind, &item.Content, &item.Revision, &active, &tombstonedAt, &tombstoneReason, &item.Scope, &item.ProvenanceJSON, &tagsRaw, &item.Importance, &createdAt, &updatedAt); err != nil {
			return result, fmt.Errorf("memory: scan goncho markdown export: %w", err)
		}
		item.State = gonchoMemoryV1StateActive
		if active == 0 {
			item.State = gonchoMemoryV1StateTombstoned
		}
		item.CreatedAt = time.Unix(createdAt, 0).UTC().Format(time.RFC3339)
		item.UpdatedAt = time.Unix(updatedAt, 0).UTC().Format(time.RFC3339)
		if tombstonedAt.Valid {
			item.TombstonedAt = time.Unix(tombstonedAt.Int64, 0).UTC().Format(time.RFC3339)
		}
		if tombstoneReason.Valid {
			item.TombstoneReason = tombstoneReason.String
		}
		_ = json.Unmarshal([]byte(tagsRaw), &item.Tags)
		doc.Items = append(doc.Items, item)
	}
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("memory: goncho markdown export rows: %w", err)
	}
	rendered, err := RenderGonchoMemoryV1Markdown(doc)
	if err != nil {
		return result, err
	}
	if err := os.MkdirAll(filepath.Dir(s.Config.Path), 0o750); err != nil {
		return result, fmt.Errorf("memory: prepare goncho markdown export dir: %w", err)
	}
	if err := os.WriteFile(s.Config.Path, []byte(rendered), 0o600); err != nil {
		return result, fmt.Errorf("memory: write goncho markdown export: %w", err)
	}
	result.Exported = len(doc.Items)
	return result, nil
}

func normalizeGonchoMarkdownReloadItem(item GonchoMemoryV1Item) GonchoMemoryV1Item {
	item.MemoryID = strings.TrimSpace(item.MemoryID)
	item.AgentID = strings.TrimSpace(item.AgentID)
	item.WorkspaceID = strings.TrimSpace(item.WorkspaceID)
	item.PeerID = strings.TrimSpace(item.PeerID)
	item.SessionID = strings.TrimSpace(item.SessionID)
	item.Scope = strings.TrimSpace(item.Scope)
	if item.Scope == "" {
		item.Scope = gonchoMemoryV1PrivateScope
	}
	item.State = strings.TrimSpace(item.State)
	if item.State == "" {
		item.State = gonchoMemoryV1StateActive
	}
	item.SourceKind = strings.TrimSpace(item.SourceKind)
	if item.SourceKind == "" {
		item.SourceKind = "manual"
	}
	item.Content = strings.Trim(item.Content, "\n")
	item.Checksum = strings.TrimSpace(item.Checksum)
	return item
}

func gonchoMarkdownUnixTime(value string) (int64, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return 0, fmt.Errorf("memory: parse goncho memory timestamp %q: %w", value, err)
	}
	return parsed.Unix(), nil
}

func gonchoMarkdownNullableUnixTime(value string) (sql.NullInt64, error) {
	if value == "" {
		return sql.NullInt64{}, nil
	}
	parsed, err := gonchoMarkdownUnixTime(value)
	if err != nil {
		return sql.NullInt64{}, err
	}
	return sql.NullInt64{Int64: parsed, Valid: true}, nil
}
