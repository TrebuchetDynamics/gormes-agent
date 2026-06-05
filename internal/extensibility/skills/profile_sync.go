package skills

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/lifecycle"
)

const (
	SkillProfileSyncConflict       = lifecycle.SkillProfileSyncConflict
	SkillProfileSyncWriteFailed    = lifecycle.SkillProfileSyncWriteFailed
	SkillProfileSyncUnavailable    = lifecycle.SkillProfileSyncUnavailable
	SkillProfileSyncInvalidProfile = lifecycle.SkillProfileSyncInvalidProfile
)

type SkillProfileRoot = lifecycle.SkillProfileRoot

type BundledSkillProfileSyncRequest = lifecycle.BundledSkillProfileSyncRequest

type BundledSkillProfileSyncReport = lifecycle.BundledSkillProfileSyncReport

type SkillProfileSyncSummary = lifecycle.SkillProfileSyncSummary

type SkillProfileSyncEvidence = lifecycle.SkillProfileSyncEvidence

func SyncBundledSkillsToProfiles(ctx context.Context, req BundledSkillProfileSyncRequest) (BundledSkillProfileSyncReport, error) {
	return lifecycle.SyncBundledSkillsToProfiles(ctx, req)
}
