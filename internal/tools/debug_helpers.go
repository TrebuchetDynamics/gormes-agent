package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/debuglog"

const (
	DebugEvidenceDisabled       = debuglog.DebugEvidenceDisabled
	DebugEvidenceLogUnavailable = debuglog.DebugEvidenceLogUnavailable
)

// DebugSessionConfig configures a per-tool debug log session. The injectable
// seams keep tests hermetic and keep production callers from reading live state
// unless they explicitly opt in through the tool-specific environment variable.
type DebugSessionConfig = debuglog.DebugSessionConfig

// DebugSessionInfo is the bounded external summary for a debug session.
type DebugSessionInfo = debuglog.DebugSessionInfo

// DebugSessionSaveResult reports whether a debug log persisted or degraded.
type DebugSessionSaveResult = debuglog.DebugSessionSaveResult

// DebugSession records optional per-tool debug calls to a JSON log file. When
// disabled it is a cheap no-op. When enabled it sanitizes secret/content-shaped
// fields before retaining or writing evidence.
type DebugSession = debuglog.DebugSession

func NewDebugSession(cfg DebugSessionConfig) *DebugSession {
	return debuglog.NewDebugSession(cfg)
}
