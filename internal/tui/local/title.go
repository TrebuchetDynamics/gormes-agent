package local

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/local/sessionadapters"
)

func NewSessionTitleFunc(ctx context.Context, smap *session.BoltMap) tui.SessionTitleFunc {
	return sessionadapters.NewSessionTitleFunc(ctx, smap)
}
