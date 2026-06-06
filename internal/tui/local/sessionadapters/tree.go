package sessionadapters

import (
	"context"

	sessionpkg "github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/local/sessionadapters/treeops"
)

func NewSessionTreeFunc(rootCtx context.Context, metadata *sessionpkg.BoltMap) tui.SessionTreeFunc {
	return treeops.NewSessionTreeFunc(rootCtx, metadata)
}

func NewSessionTreeLabelFunc(rootCtx context.Context, metadata *sessionpkg.BoltMap) tui.SessionTreeLabelFunc {
	return treeops.NewSessionTreeLabelFunc(rootCtx, metadata)
}

func NewSessionTreeRestoreFunc(rootCtx context.Context) tui.SessionTreeRestoreFunc {
	return treeops.NewSessionTreeRestoreFunc(rootCtx)
}
