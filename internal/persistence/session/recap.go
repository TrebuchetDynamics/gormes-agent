package session

import (
	"context"
	"fmt"

	recappkg "github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session/recap"
)

type RecapConfig = recappkg.Config

type RecapEntry = recappkg.Entry

type RecapEnvelope = recappkg.Envelope

type SessionRecapResult = recappkg.SessionResult

func GenerateRecap(ctx context.Context, store Map, cfg RecapConfig) (*RecapEnvelope, error) {
	if cfg.MaxEntries <= 0 {
		cfg.MaxEntries = 10
	}

	lister, ok := store.(interface {
		ListAllMetadata(ctx context.Context) ([]Metadata, error)
	})
	if !ok {
		return &RecapEnvelope{}, nil
	}

	items, err := lister.ListAllMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("recap: list sessions: %w", err)
	}

	total := len(items)
	entries := make([]RecapEntry, 0, len(items))
	limit := cfg.MaxEntries
	if limit > total {
		limit = total
	}

	for i := 0; i < limit; i++ {
		m := items[i]
		entries = append(entries, RecapEntry{
			SessionID: m.SessionID,
			Title:     m.Title,
			Source:    m.Source,
			UserID:    m.UserID,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
			TokensIn:  m.TokensInTotal,
			TokensOut: m.TokensOutTotal,
		})
	}

	return &RecapEnvelope{
		Entries:       entries,
		TotalSessions: total,
		Truncated:     total > cfg.MaxEntries,
	}, nil
}

func GenerateSessionRecap(ctx context.Context, store Map, sessionID string, cfg RecapConfig) (*SessionRecapResult, error) {
	getter, ok := store.(interface {
		GetMetadata(ctx context.Context, sessionID string) (Metadata, bool, error)
	})
	if !ok {
		return &SessionRecapResult{SessionID: sessionID, NotFound: true}, nil
	}

	meta, found, err := getter.GetMetadata(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("recap: get session %q: %w", sessionID, err)
	}
	if !found {
		return &SessionRecapResult{
			SessionID: sessionID,
			NotFound:  true,
		}, nil
	}

	return &SessionRecapResult{
		SessionID: meta.SessionID,
		Title:     meta.Title,
		Source:    meta.Source,
		UserID:    meta.UserID,
		CreatedAt: meta.CreatedAt,
		UpdatedAt: meta.UpdatedAt,
		TokensIn:  meta.TokensInTotal,
		TokensOut: meta.TokensOutTotal,
		NotFound:  false,
	}, nil
}
