package save

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/transcript"
)

const ExportTimeout = 30 * time.Second

// ExportFunc is the injection point for the TUI /save command.
type ExportFunc func(ctx context.Context, sessionID string) (path string, err error)

type RemoveFunc func(string) error

// HandleSlash preserves /save's local command semantics without coupling the
// persisted-transcript status logic to the root Bubble Tea model. It always
// returns the status line for a consumed /save invocation.
func HandleSlash(hasConversation bool, sessionID string, export ExportFunc, remove RemoveFunc) string {
	if !hasConversation {
		return "save: no conversation"
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "save: no active session"
	}
	if export == nil {
		return "save: store unavailable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), ExportTimeout)
	defer cancel()
	path, err := export(ctx, sessionID)
	if err != nil {
		if path != "" && remove != nil {
			_ = remove(path)
		}
		return FailureStatus(err)
	}
	return SuccessStatus(path)
}

func FailureStatus(err error) string {
	if errors.Is(err, transcript.ErrSessionNotFound) {
		return "save: store unavailable"
	}
	return fmt.Sprintf("save: write failed: %v", err)
}

func SuccessStatus(path string) string {
	return fmt.Sprintf("save: wrote %s", path)
}
