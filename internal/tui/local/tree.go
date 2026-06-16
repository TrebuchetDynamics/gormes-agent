package local

import (
	"context"

	sessionpkg "github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/local/sessionadapters"
)

func NewSessionTreeFunc(rootCtx context.Context, metadata *sessionpkg.BoltMap) tui.SessionTreeFunc {
	return sessionadapters.NewSessionTreeFunc(rootCtx, metadata)
}

func NewSessionTreeLabelFunc(rootCtx context.Context, metadata *sessionpkg.BoltMap) tui.SessionTreeLabelFunc {
	return sessionadapters.NewSessionTreeLabelFunc(rootCtx, metadata)
}

func NewSessionTreeRestoreFunc(rootCtx context.Context) tui.SessionTreeRestoreFunc {
	return sessionadapters.NewSessionTreeRestoreFunc(rootCtx)
}
