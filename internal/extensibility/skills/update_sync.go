package skills

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills/lifecycle"
)

const (
	SkillProfileSyncOrphaned = lifecycle.SkillProfileSyncOrphaned
)

type BundledSkillManifestEntry = lifecycle.BundledSkillManifestEntry

type BundledSkillManifestSyncRequest = lifecycle.BundledSkillManifestSyncRequest

func SyncBundledSkillsFromManifest(ctx context.Context, req BundledSkillManifestSyncRequest) (BundledSkillProfileSyncReport, error) {
	return lifecycle.SyncBundledSkillsFromManifest(ctx, req)
}
