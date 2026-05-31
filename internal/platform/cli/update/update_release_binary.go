package update

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/update/releasebinary"
)

const (
	UpdateEvidenceReleaseDownloadCompleted  UpdateEvidenceKind = UpdateEvidenceKind(releasebinary.UpdateEvidenceReleaseDownloadCompleted)
	UpdateEvidenceReleaseChecksumVerified   UpdateEvidenceKind = UpdateEvidenceKind(releasebinary.UpdateEvidenceReleaseChecksumVerified)
	UpdateEvidenceReleaseChecksumFailed     UpdateEvidenceKind = UpdateEvidenceKind(releasebinary.UpdateEvidenceReleaseChecksumFailed)
	UpdateEvidenceReleaseProvenanceVerified UpdateEvidenceKind = UpdateEvidenceKind(releasebinary.UpdateEvidenceReleaseProvenanceVerified)
	UpdateEvidenceReleaseProvenanceFailed   UpdateEvidenceKind = UpdateEvidenceKind(releasebinary.UpdateEvidenceReleaseProvenanceFailed)
	UpdateEvidenceReleaseSnapshotCreated    UpdateEvidenceKind = UpdateEvidenceKind(releasebinary.UpdateEvidenceReleaseSnapshotCreated)
	UpdateEvidenceReleaseSmokePassed        UpdateEvidenceKind = UpdateEvidenceKind(releasebinary.UpdateEvidenceReleaseSmokePassed)
	UpdateEvidenceReleaseSmokeFailed        UpdateEvidenceKind = UpdateEvidenceKind(releasebinary.UpdateEvidenceReleaseSmokeFailed)
	UpdateEvidenceReleaseSwapCompleted      UpdateEvidenceKind = UpdateEvidenceKind(releasebinary.UpdateEvidenceReleaseSwapCompleted)
	UpdateEvidenceReleaseSwapFailed         UpdateEvidenceKind = UpdateEvidenceKind(releasebinary.UpdateEvidenceReleaseSwapFailed)
	UpdateEvidenceReleaseRollbackCompleted  UpdateEvidenceKind = UpdateEvidenceKind(releasebinary.UpdateEvidenceReleaseRollbackCompleted)
	UpdateEvidenceReleaseRollbackFailed     UpdateEvidenceKind = UpdateEvidenceKind(releasebinary.UpdateEvidenceReleaseRollbackFailed)
)

type UpdateReleaseArtifact = releasebinary.UpdateReleaseArtifact
type UpdateReleaseArtifactDownloader = releasebinary.UpdateReleaseArtifactDownloader
type UpdateReleaseProvenanceVerifier = releasebinary.UpdateReleaseProvenanceVerifier
type UpdateReleaseBinaryOptions = releasebinary.UpdateReleaseBinaryOptions
type UpdateReleaseRollbackOptions = releasebinary.UpdateReleaseRollbackOptions

type UpdateReleaseBinaryReport struct {
	Failed           bool
	SnapshotID       string
	SnapshotPath     string
	PreviousVersion  string
	NewVersion       string
	ManagedBinPath   string
	PublishedBinPath string
	Evidence         []UpdateEvidence
	OperatorRecovery string
}

func (r *UpdateReleaseBinaryReport) add(kind UpdateEvidenceKind, detail string) {
	r.Evidence = append(r.Evidence, UpdateEvidence{Kind: kind, Detail: detail})
}

func RunUpdateReleaseBinaryUpdate(ctx context.Context, opts UpdateReleaseBinaryOptions) UpdateReleaseBinaryReport {
	return fromReleaseBinaryReport(releasebinary.RunUpdateReleaseBinaryUpdate(ctx, opts))
}

func RunUpdateReleaseRollback(ctx context.Context, opts UpdateReleaseRollbackOptions) UpdateReleaseBinaryReport {
	return fromReleaseBinaryReport(releasebinary.RunUpdateReleaseRollback(ctx, opts))
}

func fromReleaseBinaryReport(src releasebinary.UpdateReleaseBinaryReport) UpdateReleaseBinaryReport {
	evidence := make([]UpdateEvidence, 0, len(src.Evidence))
	for _, ev := range src.Evidence {
		evidence = append(evidence, UpdateEvidence{Kind: UpdateEvidenceKind(ev.Kind), Detail: ev.Detail})
	}
	return UpdateReleaseBinaryReport{
		Failed:           src.Failed,
		SnapshotID:       src.SnapshotID,
		SnapshotPath:     src.SnapshotPath,
		PreviousVersion:  src.PreviousVersion,
		NewVersion:       src.NewVersion,
		ManagedBinPath:   src.ManagedBinPath,
		PublishedBinPath: src.PublishedBinPath,
		Evidence:         evidence,
		OperatorRecovery: src.OperatorRecovery,
	}
}
