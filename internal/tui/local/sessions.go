package local

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/local/sessionadapters"
)

func NewSessionDirectoryFunc(ctx context.Context) tui.SessionDirectoryFunc {
	return sessionadapters.NewSessionDirectoryFunc(ctx)
}

func NewSessionResumeFunc(rootCtx context.Context, resume func(string, []llm.Message) error) tui.SessionResumeFunc {
	return sessionadapters.NewSessionResumeFunc(rootCtx, resume)
}
