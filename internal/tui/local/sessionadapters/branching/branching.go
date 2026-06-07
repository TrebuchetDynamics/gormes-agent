package branching

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	sessionpkg "github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/transcript"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/local/sessionadapters/sessiondb"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/local/sessionadapters/switching"
)

type BranchIDFunc func() string

func NewSessionBranchFunc(rootCtx context.Context, metadata sessionpkg.MetadataWriter, resume func(string, []llm.Message) error) tui.SessionBranchFunc {
	return NewSessionBranchFuncWithID(rootCtx, metadata, resume, NewSessionBranchID)
}

func NewSessionBranchFuncWithID(rootCtx context.Context, metadata sessionpkg.MetadataWriter, resume func(string, []llm.Message) error, newID BranchIDFunc) tui.SessionBranchFunc {
	switcher := switching.NewResidentSessionSwitcher(rootCtx, resume)
	return func(ctx context.Context, req tui.BranchRequest) (tui.BranchResult, error) {
		ctx = switcher.Context(ctx)
		if err := switcher.RequireResume(); err != nil {
			return tui.BranchResult{}, err
		}
		if metadata == nil {
			return tui.BranchResult{}, fmt.Errorf("metadata store unavailable")
		}
		childID := strings.TrimSpace(newID())
		if childID == "" {
			return tui.BranchResult{}, fmt.Errorf("child session id unavailable")
		}
		db, err := sessiondb.OpenDirectory()
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

		if _, err := switcher.SwitchWithTranscriptDB(ctx, db, forked.SessionID, req.History); err != nil {
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

func NewSessionBranchID() string {
	return "branch-" + strings.ReplaceAll(uuid.NewString(), "-", "")
}
