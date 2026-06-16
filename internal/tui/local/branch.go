package local

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	sessionpkg "github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/local/sessionadapters"
)

type BranchIDFunc = sessionadapters.BranchIDFunc

func NewSessionBranchFunc(rootCtx context.Context, metadata sessionpkg.MetadataWriter, resume func(string, []llm.Message) error) tui.SessionBranchFunc {
	return sessionadapters.NewSessionBranchFunc(rootCtx, metadata, resume)
}

func NewSessionBranchFuncWithID(rootCtx context.Context, metadata sessionpkg.MetadataWriter, resume func(string, []llm.Message) error, newID BranchIDFunc) tui.SessionBranchFunc {
	return sessionadapters.NewSessionBranchFuncWithID(rootCtx, metadata, resume, newID)
}

func NewSessionBranchID() string {
	return sessionadapters.NewSessionBranchID()
}
