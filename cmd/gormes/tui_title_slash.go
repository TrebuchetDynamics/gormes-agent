package main

import (
	"context"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

func newTUITitleFunc(ctx context.Context, smap *session.BoltMap) tui.SessionTitleFunc {
	if smap == nil {
		return nil
	}
	return func(sessionID, title string) (tui.SessionTitleResult, error) {
		sessionID = strings.TrimSpace(sessionID)
		title = strings.TrimSpace(title)
		if title == "" {
			meta, ok, err := smap.GetMetadata(ctx, sessionID)
			if err != nil || !ok {
				return tui.SessionTitleResult{}, err
			}
			return tui.SessionTitleResult{Title: meta.Title}, nil
		}
		if err := smap.PutMetadata(ctx, session.Metadata{
			SessionID:        sessionID,
			Title:            title,
			TitleManuallySet: true,
			UpdatedAt:        time.Now().Unix(),
		}); err != nil {
			return tui.SessionTitleResult{}, err
		}
		return tui.SessionTitleResult{Title: title}, nil
	}
}
