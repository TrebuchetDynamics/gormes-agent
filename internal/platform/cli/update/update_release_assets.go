package update

import (
	"context"

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

type UpdateReleaseAssetSkillSyncReport = releaseassets.UpdateReleaseAssetSkillSyncReport

func RunUpdateReleaseAssetSkillSync(ctx context.Context, opts UpdateReleaseAssetSkillSyncOptions) UpdateReleaseAssetSkillSyncReport {
	return releaseassets.RunUpdateReleaseAssetSkillSync(ctx, opts)
}

func RunUpdateReleaseAssetSkillRollback(ctx context.Context, opts UpdateReleaseAssetSkillRollbackOptions) UpdateReleaseAssetSkillSyncReport {
	return releaseassets.RunUpdateReleaseAssetSkillRollback(ctx, opts)
}
