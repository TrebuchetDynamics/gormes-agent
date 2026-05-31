package update

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/update/releaseassets"
)

const (
	UpdateEvidenceReleaseManifestVerified       UpdateEvidenceKind = UpdateEvidenceKind(releaseassets.UpdateEvidenceReleaseManifestVerified)
	UpdateEvidenceReleaseManifestFailed         UpdateEvidenceKind = UpdateEvidenceKind(releaseassets.UpdateEvidenceReleaseManifestFailed)
	UpdateEvidenceReleaseAssetSyncCompleted     UpdateEvidenceKind = UpdateEvidenceKind(releaseassets.UpdateEvidenceReleaseAssetSyncCompleted)
	UpdateEvidenceReleaseAssetSyncFailed        UpdateEvidenceKind = UpdateEvidenceKind(releaseassets.UpdateEvidenceReleaseAssetSyncFailed)
	UpdateEvidenceReleaseSkillSyncCompleted     UpdateEvidenceKind = UpdateEvidenceKind(releaseassets.UpdateEvidenceReleaseSkillSyncCompleted)
	UpdateEvidenceReleaseSkillSyncFailed        UpdateEvidenceKind = UpdateEvidenceKind(releaseassets.UpdateEvidenceReleaseSkillSyncFailed)
	UpdateEvidenceReleaseAssetRollbackCompleted UpdateEvidenceKind = UpdateEvidenceKind(releaseassets.UpdateEvidenceReleaseAssetRollbackCompleted)
	UpdateEvidenceReleaseAssetRollbackFailed    UpdateEvidenceKind = UpdateEvidenceKind(releaseassets.UpdateEvidenceReleaseAssetRollbackFailed)
	UpdateEvidenceReleaseAssetRollbackConflict  UpdateEvidenceKind = UpdateEvidenceKind(releaseassets.UpdateEvidenceReleaseAssetRollbackConflict)
)

type UpdateReleaseManifest = releaseassets.UpdateReleaseManifest
type UpdateReleaseAssetManifestEntry = releaseassets.UpdateReleaseAssetManifestEntry
type UpdateReleaseSkillManifestEntry = releaseassets.UpdateReleaseSkillManifestEntry
type UpdateReleaseAssetSkillSyncOptions = releaseassets.UpdateReleaseAssetSkillSyncOptions
type UpdateReleaseAssetSkillRollbackOptions = releaseassets.UpdateReleaseAssetSkillRollbackOptions

type UpdateReleaseAssetSkillSyncReport struct {
	Failed         bool
	SnapshotID     string
	SnapshotPath   string
	Evidence       []UpdateEvidence
	SkillSummaries []skills.SkillProfileSyncSummary
}

func (r *UpdateReleaseAssetSkillSyncReport) add(kind UpdateEvidenceKind, detail string) {
	r.Evidence = append(r.Evidence, UpdateEvidence{Kind: kind, Detail: detail})
}

func RunUpdateReleaseAssetSkillSync(ctx context.Context, opts UpdateReleaseAssetSkillSyncOptions) UpdateReleaseAssetSkillSyncReport {
	return fromReleaseAssetSkillSyncReport(releaseassets.RunUpdateReleaseAssetSkillSync(ctx, opts))
}

func RunUpdateReleaseAssetSkillRollback(ctx context.Context, opts UpdateReleaseAssetSkillRollbackOptions) UpdateReleaseAssetSkillSyncReport {
	return fromReleaseAssetSkillSyncReport(releaseassets.RunUpdateReleaseAssetSkillRollback(ctx, opts))
}

func fromReleaseAssetSkillSyncReport(src releaseassets.UpdateReleaseAssetSkillSyncReport) UpdateReleaseAssetSkillSyncReport {
	evidence := make([]UpdateEvidence, 0, len(src.Evidence))
	for _, ev := range src.Evidence {
		evidence = append(evidence, UpdateEvidence{Kind: UpdateEvidenceKind(ev.Kind), Detail: ev.Detail})
	}
	return UpdateReleaseAssetSkillSyncReport{
		Failed:         src.Failed,
		SnapshotID:     src.SnapshotID,
		SnapshotPath:   src.SnapshotPath,
		Evidence:       evidence,
		SkillSummaries: src.SkillSummaries,
	}
}
