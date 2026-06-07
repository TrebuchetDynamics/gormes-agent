package snapshot

import "strings"

// Mode controls whether the terminal-environment shell wrapper sources a
// persisted bash environment snapshot before running a user command.
type Mode int

const (
	// Disabled returns the user command verbatim. No snapshot loading is emitted.
	Disabled Mode = iota
	// Enabled prefixes the user command with a silenced `source <path>` line.
	Enabled
)

// Evidence codes for Evidence.Code.
const (
	EvidenceLoaded      = "snapshot_loaded"
	EvidenceDisabled    = "snapshot_disabled"
	EvidencePathMissing = "snapshot_path_missing"
)

// Config configures BuildShellWrapper.
type Config struct {
	// Mode selects whether to emit the snapshot-source prefix.
	Mode Mode
	// SnapshotPath is the absolute path to the persisted bash environment dump.
	SnapshotPath string
}

// Evidence is telemetry about wrapper construction.
type Evidence struct {
	// Code is one of EvidenceLoaded, EvidenceDisabled, or EvidencePathMissing.
	Code string
	// Path echoes the requested snapshot path so callers can record it.
	Path string
}

// BuildShellWrapper returns a shell-script string that, when enabled, loads a
// persisted environment snapshot with stdout AND stderr redirected to
// /dev/null before running the user command verbatim.
func BuildShellWrapper(cfg Config, userCommand string) (string, Evidence) {
	if cfg.Mode != Enabled {
		return userCommand, Evidence{
			Code: EvidenceDisabled,
			Path: cfg.SnapshotPath,
		}
	}
	if cfg.SnapshotPath == "" {
		return userCommand, Evidence{
			Code: EvidencePathMissing,
			Path: "",
		}
	}

	// Source the snapshot with both stdout and stderr redirected, then run the
	// user command verbatim on the next line. `|| true` keeps the wrapper
	// resilient to a missing/corrupt snapshot file.
	sourceLine := "source " + shellQuoteSnapshotPath(cfg.SnapshotPath) + " >/dev/null 2>&1 || true"
	wrapper := sourceLine + "\n" + userCommand
	return wrapper, Evidence{
		Code: EvidenceLoaded,
		Path: cfg.SnapshotPath,
	}
}

func shellQuoteSnapshotPath(path string) string {
	return "'" + strings.ReplaceAll(path, "'", "'\"'\"'") + "'"
}
