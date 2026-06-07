package environment

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/environment/snapshot"

// SnapshotMode controls whether the terminal-environment shell wrapper
// sources a persisted bash environment snapshot before running a user command.
type SnapshotMode = snapshot.Mode

const (
	// SnapshotDisabled returns the user command verbatim. No snapshot loading is emitted.
	SnapshotDisabled SnapshotMode = snapshot.Disabled
	// SnapshotEnabled prefixes the user command with a silenced source line.
	SnapshotEnabled SnapshotMode = snapshot.Enabled
)

// Evidence codes for EnvironmentSnapshotEvidence.Code.
const (
	EvidenceSnapshotLoaded      = snapshot.EvidenceLoaded
	EvidenceSnapshotDisabled    = snapshot.EvidenceDisabled
	EvidenceSnapshotPathMissing = snapshot.EvidencePathMissing
)

// EnvironmentSnapshotConfig configures BuildShellWrapper.
type EnvironmentSnapshotConfig = snapshot.Config

// EnvironmentSnapshotEvidence is telemetry about wrapper construction.
type EnvironmentSnapshotEvidence = snapshot.Evidence

// BuildShellWrapper returns a shell-script string that, when enabled, loads a
// persisted environment snapshot with stdout AND stderr redirected to
// /dev/null before running the user command verbatim.
func BuildShellWrapper(cfg EnvironmentSnapshotConfig, userCommand string) (string, EnvironmentSnapshotEvidence) {
	return snapshot.BuildShellWrapper(cfg, userCommand)
}
