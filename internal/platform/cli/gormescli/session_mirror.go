package gormescli

import (
	"log/slog"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

const sessionIndexMirrorInterval = 30 * time.Second

func StartSessionIndexMirror(smap *session.BoltMap, log *slog.Logger) *session.SessionIndexMirrorRefresher {
	if smap == nil {
		return nil
	}
	mirror := session.NewSessionIndexMirror(smap, config.SessionIndexMirrorPath())
	return mirror.StartRefresh(sessionIndexMirrorInterval, log)
}
