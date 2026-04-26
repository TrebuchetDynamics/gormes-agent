package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/transcript"
)

// saveExportTimeout caps how long /save waits on the injected helper. The
// helper writes a markdown file backed by a single SQLite read; thirty
// seconds is generous, and exceeding it surfaces a status line instead of
// blocking the editor on a stuck disk.
const saveExportTimeout = 30 * time.Second

// SessionExportFunc is the injection point for the TUI /save command. The
// implementation produced by cmd/gormes opens config.MemoryDBPath(), calls
// internal/transcript.ExportMarkdown to build the canonical persisted
// transcript, writes the result to disk, and returns the file path. Tests
// wire fakes; the unit tests in this package never open a real DB.
//
// The returned path may be non-empty even when err != nil. The slash
// handler interprets that combination as "writer left a partial file
// behind" and removes it via os.Remove before reporting the failure to the
// user, so a half-written transcript is never visible to the operator.
type SessionExportFunc func(ctx context.Context, sessionID string) (path string, err error)

// saveSlashHandler implements /save. It MUST consume the input on every
// branch (Handled=true) so the slash text never falls through to
// kernel.Submit, and it must NEVER write UI-only transcripts directly —
// the canonical persisted-store reader (transcript.ExportMarkdown) is the
// only sanctioned source, accessed through the injected SessionExportFunc.
func saveSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "save: store unavailable"}
	}
	if len(model.frame.History) == 0 {
		return SlashResult{Handled: true, StatusMessage: "save: no conversation"}
	}
	sessionID := strings.TrimSpace(model.frame.SessionID)
	if sessionID == "" {
		return SlashResult{Handled: true, StatusMessage: "save: no active session"}
	}
	if model.sessionExport == nil {
		return SlashResult{Handled: true, StatusMessage: "save: store unavailable"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), saveExportTimeout)
	defer cancel()
	path, err := model.sessionExport(ctx, sessionID)
	if err != nil {
		if path != "" {
			_ = os.Remove(path)
		}
		if errors.Is(err, transcript.ErrSessionNotFound) {
			return SlashResult{Handled: true, StatusMessage: "save: store unavailable"}
		}
		return SlashResult{Handled: true, StatusMessage: fmt.Sprintf("save: write failed: %v", err)}
	}
	return SlashResult{
		Handled:       true,
		StatusMessage: fmt.Sprintf("save: wrote %s", path),
	}
}
