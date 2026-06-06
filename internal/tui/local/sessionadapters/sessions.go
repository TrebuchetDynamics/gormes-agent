package sessionadapters

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/local/sessionadapters/directory"
)

func NewSessionDirectoryFunc(ctx context.Context) tui.SessionDirectoryFunc {
	return directory.NewSessionDirectoryFunc(ctx)
}

func NewSessionResumeFunc(rootCtx context.Context, resume func(string, []llm.Message) error) tui.SessionResumeFunc {
	return directory.NewSessionResumeFunc(rootCtx, resume)
}
