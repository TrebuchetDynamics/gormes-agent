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

type UpdateReleaseBinaryReport = releasebinary.UpdateReleaseBinaryReport

func addUpdateReleaseBinaryEvidence(r *UpdateReleaseBinaryReport, kind UpdateEvidenceKind, detail string) {
	r.Evidence = append(r.Evidence, UpdateEvidence{Kind: kind, Detail: detail})
}

func RunUpdateReleaseBinaryUpdate(ctx context.Context, opts UpdateReleaseBinaryOptions) UpdateReleaseBinaryReport {
	return releasebinary.RunUpdateReleaseBinaryUpdate(ctx, opts)
}

func RunUpdateReleaseRollback(ctx context.Context, opts UpdateReleaseRollbackOptions) UpdateReleaseBinaryReport {
	return releasebinary.RunUpdateReleaseRollback(ctx, opts)
}
