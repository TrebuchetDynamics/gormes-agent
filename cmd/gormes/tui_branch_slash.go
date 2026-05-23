package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	sessionpkg "github.com/TrebuchetDynamics/gormes-agent/internal/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/transcript"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
)

type tuiBranchIDFunc func() string

func newTUIBranchFunc(rootCtx context.Context, metadata sessionpkg.MetadataWriter, resume func(string, []hermes.Message) error) tui.SessionBranchFunc {
	return newTUIBranchFuncWithID(rootCtx, metadata, resume, newTUIBranchSessionID)
}

func newTUIBranchFuncWithID(rootCtx context.Context, metadata sessionpkg.MetadataWriter, resume func(string, []hermes.Message) error, newID tuiBranchIDFunc) tui.SessionBranchFunc {
	switcher := newTUIResidentSessionSwitcher(rootCtx, resume)
	return func(ctx context.Context, req tui.BranchRequest) (tui.BranchResult, error) {
		ctx = switcher.context(ctx)
		if err := switcher.requireResume(); err != nil {
			return tui.BranchResult{}, err
		}
		if metadata == nil {
			return tui.BranchResult{}, fmt.Errorf("metadata store unavailable")
		}
		childID := strings.TrimSpace(newID())
		if childID == "" {
			return tui.BranchResult{}, fmt.Errorf("child session id unavailable")
		}
		db, err := openSessionDirectoryDB()
		if err != nil {
			return tui.BranchResult{}, err
		}
		defer db.Close()

		forked, err := sessionpkg.Fork(ctx, metadata, transcriptTurnCopier{db: db}, sessionpkg.ForkRequest{
			ParentSessionID: req.ParentSessionID,
			ChildSessionID:  childID,
			Title:           req.Title,
		})
		if err != nil {
			return tui.BranchResult{}, err
		}

		if _, err := switcher.switchWithTranscriptDB(ctx, db, forked.SessionID, req.History); err != nil {
			return tui.BranchResult{}, err
		}

		return tui.BranchResult{
			SessionID:        forked.SessionID,
			ParentSessionID:  forked.ParentSessionID,
			Title:            forked.Title,
			TranscriptCopied: forked.TranscriptCopied,
		}, nil
	}
}

type transcriptTurnCopier struct {
	db *sql.DB
}

func (c transcriptTurnCopier) CopyTurns(ctx context.Context, parentSessionID, childSessionID string) (int, error) {
	return transcript.ForkTurns(ctx, c.db, parentSessionID, childSessionID)
}

func newTUIBranchSessionID() string {
	return "branch-" + strings.ReplaceAll(uuid.NewString(), "-", "")
}
