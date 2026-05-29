package session

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type RecapConfig struct {
	MaxEntries int
}

type RecapEntry struct {
	SessionID string
	Title     string
	Source    string
	UserID    string
	CreatedAt int64
	UpdatedAt int64
	TokensIn  int
	TokensOut int
}

type RecapEnvelope struct {
	Entries       []RecapEntry
	TotalSessions int
	Truncated     bool
}

func (e *RecapEnvelope) HumanOutput() string {
	if len(e.Entries) == 0 {
		return "No sessions found."
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Sessions (%d total", e.TotalSessions))
	if e.Truncated {
		sb.WriteString(fmt.Sprintf(", showing %d", len(e.Entries)))
	}
	sb.WriteString("):\n\n")

	for _, entry := range e.Entries {
		title := entry.Title
		if title == "" {
			title = "(untitled)"
		}
		sb.WriteString(fmt.Sprintf("  %-20s  %s  [%s]\n", entry.SessionID, title, entry.Source))
		if entry.TokensIn > 0 || entry.TokensOut > 0 {
			sb.WriteString(fmt.Sprintf("    tokens: %d in / %d out\n", entry.TokensIn, entry.TokensOut))
		}
		if entry.UpdatedAt > 0 {
			sb.WriteString(fmt.Sprintf("    updated: %s\n", time.Unix(entry.UpdatedAt, 0).Format(time.RFC3339)))
		}
	}

	if e.Truncated {
		sb.WriteString(fmt.Sprintf("\n... %d more sessions not shown. Increase --limit to see more.\n", e.TotalSessions-len(e.Entries)))
	}

	return sb.String()
}

type SessionRecapResult struct {
	SessionID string
	Title     string
	Source    string
	UserID    string
	CreatedAt int64
	UpdatedAt int64
	TokensIn  int
	TokensOut int
	NotFound  bool
}

func (r *SessionRecapResult) HumanOutput() string {
	if r.NotFound {
		return fmt.Sprintf("Session %q not found.", r.SessionID)
	}

	var sb strings.Builder
	title := r.Title
	if title == "" {
		title = "(untitled)"
	}
	sb.WriteString(fmt.Sprintf("Session: %s\n", r.SessionID))
	sb.WriteString(fmt.Sprintf("Title:   %s\n", title))
	sb.WriteString(fmt.Sprintf("Source:  %s\n", r.Source))
	if r.UserID != "" {
		sb.WriteString(fmt.Sprintf("User:    %s\n", r.UserID))
	}
	if r.CreatedAt > 0 {
		sb.WriteString(fmt.Sprintf("Created: %s\n", time.Unix(r.CreatedAt, 0).Format(time.RFC3339)))
	}
	if r.UpdatedAt > 0 {
		sb.WriteString(fmt.Sprintf("Updated: %s\n", time.Unix(r.UpdatedAt, 0).Format(time.RFC3339)))
	}
	if r.TokensIn > 0 || r.TokensOut > 0 {
		sb.WriteString(fmt.Sprintf("Tokens:  %d in / %d out\n", r.TokensIn, r.TokensOut))
	}

	return sb.String()
}

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
