package sessionadapters

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/local/sessionadapters/titleops"
)

func NewSessionTitleFunc(ctx context.Context, smap *session.BoltMap) tui.SessionTitleFunc {
	return titleops.NewSessionTitleFunc(ctx, smap)
}
