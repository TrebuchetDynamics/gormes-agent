package sessionadapters

import (
	"context"
	"errors"
	"fmt"
	"strings"

	appsession "github.com/TrebuchetDynamics/gormes-agent/internal/app/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	sessionpkg "github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func NewSessionDirectoryFunc(ctx context.Context) tui.SessionDirectoryFunc {
	return func(limit int) ([]tui.SessionDirectoryEntry, error) {
		if limit <= 0 {
			limit = 20
		}
		db, err := appsession.OpenSessionDirectoryDB()
		if err != nil {
			if strings.Contains(err.Error(), "memory database not found") {
				return nil, nil
			}
			return nil, err
		}
		defer db.Close()

		sessions, err := sessionpkg.ListDirectorySessions(ctx, db, sessionpkg.DirectoryFilter{Limit: limit})
		if err != nil {
			return nil, err
		}
		sessions = appsession.ApplySessionMirrorSources(sessions, config.SessionIndexMirrorPath())
		out := make([]tui.SessionDirectoryEntry, 0, len(sessions))
		for _, item := range sessions {
			out = append(out, tui.SessionDirectoryEntry{
				ID:           item.ID,
				Title:        item.Title,
				Preview:      item.Preview,
				Source:       item.Source,
				LastActiveAt: item.LastActiveAt,
				MessageCount: item.MessageCount,
			})
		}
		return out, nil
	}
}

func NewSessionResumeFunc(rootCtx context.Context, resume func(string, []llm.Message) error) tui.SessionResumeFunc {
	switcher := newResidentSessionSwitcher(rootCtx, resume)
	return func(ctx context.Context, query string) (tui.SessionResumeResult, error) {
		ctx = switcher.context(ctx)
		query = strings.TrimSpace(query)
		if query == "" {
			return tui.SessionResumeResult{}, fmt.Errorf("session id or prefix required")
		}
		if err := switcher.requireResume(); err != nil {
			return tui.SessionResumeResult{}, err
		}
		db, err := appsession.OpenSessionDirectoryDB()
		if err != nil {
			return tui.SessionResumeResult{}, err
		}
		defer db.Close()

		resolved, err := sessionpkg.ResolveSessionIDPrefix(ctx, db, query)
		if err != nil {
			if errors.Is(err, sessionpkg.ErrSessionNotFound) {
				return tui.SessionResumeResult{}, fmt.Errorf("session %q not found", query)
			}
			if errors.Is(err, sessionpkg.ErrSessionPrefixAmbiguous) {
				return tui.SessionResumeResult{}, fmt.Errorf("session prefix %q is ambiguous: %w", query, err)
			}
			return tui.SessionResumeResult{}, err
		}

		history, err := switcher.switchWithTranscriptDB(ctx, db, resolved, nil)
		if err != nil {
			return tui.SessionResumeResult{}, err
		}
		return tui.SessionResumeResult{SessionID: resolved, History: history}, nil
	}
}
