package environment

import "strings"

// SnapshotMode controls whether the terminal-environment shell wrapper
// sources a persisted bash environment snapshot before running a user
// command. The wrapper exists so future Goncho/Hermes terminal backends can
// preserve environment variables across tool invocations without leaking the
// snapshot's `declare -x` lines into the model-visible output (Hermes commit
// 2e6699b3, tools/environments/base.py::BaseEnvironment._wrap_command).
type SnapshotMode int

const (
	// SnapshotDisabled returns the user command verbatim. No snapshot
	// loading is emitted.
	SnapshotDisabled SnapshotMode = iota
	// SnapshotEnabled prefixes the user command with a silenced
	// `source <path>` line that suppresses both stdout and stderr.
	SnapshotEnabled
)

// Evidence codes for EnvironmentSnapshotEvidence.Code.
const (
	EvidenceSnapshotLoaded      = "snapshot_loaded"
	EvidenceSnapshotDisabled    = "snapshot_disabled"
	EvidenceSnapshotPathMissing = "snapshot_path_missing"
)

// EnvironmentSnapshotConfig configures BuildShellWrapper.
type EnvironmentSnapshotConfig struct {
	// Mode selects whether to emit the snapshot-source prefix.
	Mode SnapshotMode
	// SnapshotPath is the absolute path to the persisted bash environment
	// dump (typically the output of `export -p` plus filtered functions
	// and aliases). Empty paths fall back to disabled behavior.
	SnapshotPath string
}

// EnvironmentSnapshotEvidence is telemetry about wrapper construction.
type EnvironmentSnapshotEvidence struct {
	// Code is one of EvidenceSnapshotLoaded, EvidenceSnapshotDisabled, or
	// EvidenceSnapshotPathMissing.
	Code string
	// Path echoes the requested snapshot path so callers can record it
	// even when the wrapper falls through.
	Path string
}

// BuildShellWrapper returns a shell-script string that, when enabled, loads a
// persisted environment snapshot with stdout AND stderr redirected to
// /dev/null before running the user command verbatim. macOS bash 3.2 emits
// `declare -x` lines to stdout when sourcing such a file, which would leak
// the developer's environment variables into every tool response unless both
// streams are suppressed (Hermes 2e6699b3 fix).
//
// When Mode is SnapshotDisabled or SnapshotPath is empty, the user command
// is returned untouched and the evidence reports the fall-through reason.
//
// The user command is never wrapped in its own /dev/null redirect — only the
// snapshot-load line is silenced so the actual command output remains
// visible to the caller.
func BuildShellWrapper(cfg EnvironmentSnapshotConfig, userCommand string) (string, EnvironmentSnapshotEvidence) {
	if cfg.Mode != SnapshotEnabled {
		return userCommand, EnvironmentSnapshotEvidence{
			Code: EvidenceSnapshotDisabled,
			Path: cfg.SnapshotPath,
		}
	}
	if cfg.SnapshotPath == "" {
		return userCommand, EnvironmentSnapshotEvidence{
			Code: EvidenceSnapshotPathMissing,
			Path: "",
		}
	}

	// Source the snapshot with both stdout and stderr redirected, then run
	// the user command verbatim on the next line. `|| true` keeps the
	// wrapper resilient to a missing/corrupt snapshot file — the user
	// command still runs, matching Hermes behavior.
	sourceLine := "source " + shellQuoteSnapshotPath(cfg.SnapshotPath) + " >/dev/null 2>&1 || true"
	wrapper := sourceLine + "\n" + userCommand
	return wrapper, EnvironmentSnapshotEvidence{
		Code: EvidenceSnapshotLoaded,
		Path: cfg.SnapshotPath,
	}
}

func shellQuoteSnapshotPath(path string) string {
	return "'" + strings.ReplaceAll(path, "'", "'\"'\"'") + "'"
}
