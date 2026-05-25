// Package goncho provides a thin adapter layer between gormes-agent's internal
// session types and the standalone goncho memory library.
//
// The external goncho package (github.com/TrebuchetDynamics/goncho) is the
// canonical implementation. This package exists solely to bridge type mismatches
// between gormes-agent's internal session.Metadata and goncho.SessionMetadata.
package goncho

import (
	"context"

	extgoncho "github.com/TrebuchetDynamics/goncho/service"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
)

// SessionDirectoryAdapter wraps a *session.BoltMap to satisfy the
// extgoncho.SessionDirectory interface by mapping session.Metadata fields
// to goncho.SessionMetadata.
type SessionDirectoryAdapter struct {
	Map *session.BoltMap
}

// ListMetadataByUserID implements extgoncho.SessionDirectory.
func (a *SessionDirectoryAdapter) ListMetadataByUserID(ctx context.Context, userID string) ([]extgoncho.SessionMetadata, error) {
	metas, err := a.Map.ListMetadataByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]extgoncho.SessionMetadata, len(metas))
	for i, m := range metas {
		out[i] = extgoncho.SessionMetadata{
			SessionID:       m.SessionID,
			Source:          m.Source,
			ChatID:          m.ChatID,
			UserID:          m.UserID,
			Title:           m.Title,
			ParentSessionID: m.ParentSessionID,
			LineageKind:     m.LineageKind,
			CreatedAt:       m.CreatedAt,
			UpdatedAt:       m.UpdatedAt,
		}
	}
	return out, nil
}

// ToGonchoSessionMetadata converts a single session.Metadata to goncho.SessionMetadata.
func ToGonchoSessionMetadata(m session.Metadata) extgoncho.SessionMetadata {
	return extgoncho.SessionMetadata{
		SessionID:       m.SessionID,
		Source:          m.Source,
		ChatID:          m.ChatID,
		UserID:          m.UserID,
		Title:           m.Title,
		ParentSessionID: m.ParentSessionID,
		LineageKind:     m.LineageKind,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}
