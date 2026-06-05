package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/environment"

type SnapshotMode = environment.SnapshotMode

const (
	SnapshotDisabled SnapshotMode = environment.SnapshotDisabled
	SnapshotEnabled  SnapshotMode = environment.SnapshotEnabled
)

const (
	EvidenceSnapshotLoaded      = environment.EvidenceSnapshotLoaded
	EvidenceSnapshotDisabled    = environment.EvidenceSnapshotDisabled
	EvidenceSnapshotPathMissing = environment.EvidenceSnapshotPathMissing
)

type EnvironmentSnapshotConfig = environment.EnvironmentSnapshotConfig
type EnvironmentSnapshotEvidence = environment.EnvironmentSnapshotEvidence

func BuildShellWrapper(cfg EnvironmentSnapshotConfig, userCommand string) (string, EnvironmentSnapshotEvidence) {
	return environment.BuildShellWrapper(cfg, userCommand)
}
