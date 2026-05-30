package tui

import (
	"os"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/save"
)

// saveExportTimeout caps how long /save waits on the injected helper. The
// helper writes a markdown file backed by a single SQLite read; thirty
// seconds is generous, and exceeding it surfaces a status line instead of
// blocking the editor on a stuck disk.
const saveExportTimeout = save.ExportTimeout

// SessionExportFunc is the injection point for the TUI /save command. The
// implementation produced by cmd/gormes opens config.MemoryDBPath(), calls
// internal/persistence/transcript.ExportMarkdown to build the canonical persisted
// transcript, writes the result to disk, and returns the file path. Tests
// wire fakes; the unit tests in this package never open a real DB.
//
// The returned path may be non-empty even when err != nil. The slash
// handler interprets that combination as "writer left a partial file
// behind" and removes it via os.Remove before reporting the failure to the
// user, so a half-written transcript is never visible to the operator.
type SessionExportFunc = save.ExportFunc

// saveSlashHandler implements /save. It MUST consume the input on every
// branch (Handled=true) so the slash text never falls through to
// kernel.Submit, and it must NEVER write UI-only transcripts directly —
// the canonical persisted-store reader (transcript.ExportMarkdown) is the
// only sanctioned source, accessed through the injected SessionExportFunc.
func saveSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "save: store unavailable"}
	}
	return SlashResult{Handled: true, StatusMessage: save.HandleSlash(
		len(model.frame.History) > 0,
		model.frame.SessionID,
		model.sessionExport,
		os.Remove,
	)}
}
